"""HTTP driver for scripts/e2e.sh. Uses only the standard library.

  e2e_driver.py learn  <api> <curriculum_dir>   master every ready lesson
  e2e_driver.py verify <api> <curriculum_dir>   check progress survived a restart
"""
import glob
import json
import os
import sys
import urllib.error
import urllib.request


def call(api, method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(api + path, data=data, method=method, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            return r.status, json.load(r)
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            return e.code, json.loads(raw)
        except ValueError:
            return e.code, {"raw": raw.decode(errors="replace")[:300]}


def lessons(curriculum):
    out = {}
    for p in glob.glob(os.path.join(curriculum, "lessons", "*.json")):
        with open(p) as f:
            l = json.load(f)
        out[l["nodeId"]] = l
    return out


def fail(msg):
    print("FAIL:", msg)
    sys.exit(1)


def roadmap_statuses(api):
    _, rm = call(api, "GET", "/api/v1/roadmap")
    return {n["id"]: n["status"] for n in rm["nodes"]}, rm


def drive(api, node_id, lesson):
    status, s = call(api, "POST", "/api/v1/sessions", {"nodeId": node_id})
    if status != 200:
        fail(f"start {node_id}: {status} {s}")
    prompts = lesson["guidedPractice"] + lesson["independentPractice"]
    for _ in range(40):
        if s["phase"] == "ASSESSMENT":
            break
        last = s["messages"][-1]["content"]
        answer, at = "ready", -1
        for p in prompts:
            k = last.rfind(p["prompt"])
            if k > at:
                answer, at = p["expected"][0], k
        status, s = call(api, "POST", f"/api/v1/sessions/{s['id']}/messages", {"content": answer})
        if status != 200:
            fail(f"message {node_id}: {status} {s}")
    else:
        fail(f"{node_id} never reached ASSESSMENT")

    # A failing attempt first: must not unlock anything.
    wrong = [{"itemId": it["id"], "answer": "zzz"} for it in lesson["assessment"]]
    status, r = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"answers": wrong})
    if status != 200 or r["passed"] or r["unlocked"]:
        fail(f"{node_id}: failing attempt misbehaved: {status} {r}")
    if r["session"]["phase"] != "REVIEW":
        fail(f"{node_id}: expected REVIEW after failing, got {r['session']['phase']}")
    s = r["session"]
    for _ in range(40):
        if s["phase"] == "ASSESSMENT":
            break
        last = s["messages"][-1]["content"]
        answer, at = "ready", -1
        for p in prompts:
            k = last.rfind(p["prompt"])
            if k > at:
                answer, at = p["expected"][0], k
        status, s = call(api, "POST", f"/api/v1/sessions/{s['id']}/messages", {"content": answer})
    right = [{"itemId": it["id"], "answer": it["acceptableAnswers"][0]} for it in lesson["assessment"]]
    status, r = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"answers": right})
    if status != 200 or not r["passed"]:
        fail(f"{node_id}: passing attempt failed: {status} {r}")
    # Repeated completion request is idempotent.
    status, again = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"answers": right})
    if status != 200 or not again["alreadyCompleted"] or again["newlyMastered"]:
        fail(f"{node_id}: resubmission not idempotent: {status} {again}")
    unlocked = [n["id"] for n in r["unlocked"]]
    print(f"  ✓ {node_id}: {r['attempt']['score']}/100, unlocked {unlocked or '—'}")


def learn(api, curriculum):
    ls = lessons(curriculum)
    for _ in range(len(ls) + 1):
        statuses, _ = roadmap_statuses(api)
        todo = [n for n in ls if statuses.get(n) in ("available", "in_progress")]
        if not todo:
            break
        drive(api, todo[0], ls[todo[0]])
    statuses, _ = roadmap_statuses(api)
    not_done = [n for n in ls if statuses.get(n) != "mastered"]
    if not_done:
        fail(f"not mastered: {not_done}")
    # Locked / coming-soon nodes refuse to start.
    for nid, want in (("a2-past-simple-regular", "node_coming_soon"), ("a1-self-introduction", "node_coming_soon")):
        status, body = call(api, "POST", "/api/v1/sessions", {"nodeId": nid})
        if status != 409:
            fail(f"{nid} should not start: {status} {body}")
    print(f"  ✓ all {len(ls)} ready lessons mastered")


def verify(api, curriculum):
    statuses, rm = roadmap_statuses(api)
    ls = lessons(curriculum)
    missing = [n for n in ls if statuses.get(n) != "mastered"]
    if missing:
        fail(f"progress lost after restart: {missing}")
    _, p = call(api, "GET", "/api/v1/progress")
    print(f"  ✓ after restart: {len(p['mastered'])} mastered, level {p['learner']['currentLevel']}, {len(p['recentAttempts'])} attempts on record")


if __name__ == "__main__":
    mode, api, curriculum = sys.argv[1:4]
    {"learn": learn, "verify": verify}[mode](api, curriculum)
