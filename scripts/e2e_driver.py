"""HTTP driver for scripts/e2e.sh. Uses only the standard library.

  e2e_driver.py learn  <api> <course_dir>   sign up, onboard, master every ready lesson
  e2e_driver.py verify <api> <course_dir>   log in again, check progress survived
"""
import glob
import json
import os
import sys
import urllib.error
import urllib.request


COURSE = "english"
EMAIL_A, EMAIL_B, PASSWORD = "e2e-a@example.com", "e2e-b@example.com", "e2e-password-123"
TOKEN = {"value": ""}


def call(api, method, path, body=None, lang=None):
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Content-Type": "application/json", "X-FluentPath-Client": "native"}
    if lang:
        headers["Accept-Language"] = lang
    if TOKEN["value"]:
        headers["Authorization"] = "Bearer " + TOKEN["value"]
    req = urllib.request.Request(api + path, data=data, method=method, headers=headers)
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


def sign_in(api, email, register):
    TOKEN["value"] = ""
    path = "/api/v1/auth/register" if register else "/api/v1/auth/login"
    body = {"email": email, "password": PASSWORD}
    if register:
        body["displayName"] = email.split("@")[0]
    status, r = call(api, "POST", path, body)
    if status != 200 or not r.get("token"):
        fail(f"{path} for {email}: {status} {r}")
    TOKEN["value"] = r["token"]


def roadmap_statuses(api):
    _, rm = call(api, "GET", f"/api/v1/courses/{COURSE}/roadmap")
    return {n["id"]: n["status"] for n in rm["nodes"]}, rm


def drive(api, node_id, lesson, helper=False):
    status, s = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": node_id})
    if status != 200:
        fail(f"start {node_id}: {status} {s}")
    if s["phase"] != "LEARNING" or not s["material"] or len(s["material"]["sections"]) < 3 or not s["assessment"]:
        fail(f"{node_id}: lesson should open on its material with the final quiz available: {s.get('phase')}")
    if helper:
        # The AI helper (mock): a question, a practice quiz and the lesson's videos.
        status, s = call(api, "POST", f"/api/v1/sessions/{s['id']}/messages", {"content": "Can you explain this again?"})
        if status != 200 or s["messages"][-1]["role"] != "assistant":
            fail(f"{node_id}: helper question: {status} {s}")
        status, s = call(api, "POST", f"/api/v1/sessions/{s['id']}/messages", {"intent": "quiz"})
        if status != 200 or len(s["messages"][-1]["metadata"].get("quiz", [])) < 2:
            fail(f"{node_id}: helper quiz: {status} {s['messages'][-1]}")
        status, s = call(api, "POST", f"/api/v1/sessions/{s['id']}/messages", {"intent": "video"})
        if status != 200 or not s["messages"][-1]["metadata"].get("resources"):
            fail(f"{node_id}: helper videos: {status} {s['messages'][-1]}")
        # Save the lesson's starter flashcards and review one.
        cards = s["material"]["flashcards"]
        status, r = call(api, "POST", "/api/v1/flashcards", {"course": COURSE, "nodeId": node_id, "cards": cards})
        if status != 200 or r["added"] != len(cards):
            fail(f"{node_id}: save flashcards: {status} {r}")
        status, q = call(api, "GET", "/api/v1/flashcards/due")
        if status != 200 or len(q["cards"]) != len(cards):
            fail(f"{node_id}: due flashcards: {status} {q}")
        status, c = call(api, "POST", f"/api/v1/flashcards/{q['cards'][0]['id']}/review", {"grade": "good"})
        if status != 200 or c["intervalDays"] != 1:
            fail(f"{node_id}: review flashcard: {status} {c}")
        print(f"  ✓ {node_id}: helper answered, made a quiz, showed videos; {len(cards)} flashcards saved, one reviewed")

    # A failing attempt first: must not unlock anything, and the lesson stays open.
    wrong = [{"itemId": it["id"], "answer": "zzz"} for it in lesson["assessment"]]
    status, r = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"answers": wrong})
    if status != 200 or r["passed"] or r["unlocked"]:
        fail(f"{node_id}: failing attempt misbehaved: {status} {r}")
    if r["session"]["phase"] != "LEARNING" or not r["session"]["canChat"]:
        fail(f"{node_id}: expected the lesson to stay open after failing, got {r['session']['phase']}")
    s = r["session"]
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


def onboard(api):
    status, p = call(api, "GET", f"/api/v1/courses/{COURSE}/profile")
    if status != 200 or p["completed"] or not p["questions"]:
        fail(f"fresh profile: {status} {p}")
    status, body = call(api, "PUT", f"/api/v1/courses/{COURSE}/profile", {"answers": {"goal": "moon"}})
    if status != 400 or body["error"]["code"] != "invalid_profile":
        fail(f"bad onboarding answer should be rejected: {status} {body}")
    status, rm = call(api, "PUT", f"/api/v1/courses/{COURSE}/profile",
                      {"answers": {"goal": "tech", "background": "other", "background_text": "Data analyst"}})
    if status != 200 or not rm["enrolled"] or not rm["profile"]["completed"]:
        fail(f"save profile: {status} {rm}")
    if rm["enrollment"]["targetLevel"] != "B1":
        fail(f"tech goal should target B1, got {rm['enrollment']['targetLevel']}")
    rec = {n["id"] for n in rm["nodes"] if n["recommended"]}
    if "a1-simple-present" not in rec or "a0-alphabet-sounds" in rec:
        fail(f"recommended nodes wrong: {sorted(rec)}")
    status, s = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": "a0-alphabet-sounds"})
    if status != 200 or "technical English" not in s["greeting"]:
        fail(f"the helper should greet with the goal: {status} {s.get('greeting')}")
    print(f"  ✓ onboarding: goal tech → target B1, {len(rec)} recommended skills, personal greeting")


def learn(api, curriculum):
    # Anonymous visitors can browse but not learn.
    status, _ = call(api, "GET", f"/api/v1/courses/{COURSE}/roadmap")
    if status != 200:
        fail(f"public roadmap: {status}")
    status, _ = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": "a0-alphabet-sounds"})
    if status != 401:
        fail(f"anonymous start should be 401, got {status}")
    sign_in(api, EMAIL_B, register=True)
    sign_in(api, EMAIL_A, register=True)
    onboard(api)
    ls = lessons(curriculum)
    for _ in range(len(ls) + 1):
        statuses, _ = roadmap_statuses(api)
        todo = [n for n in ls if statuses.get(n) in ("available", "in_progress")]
        if not todo:
            break
        drive(api, todo[0], ls[todo[0]], helper=todo[0] == "a0-alphabet-sounds")
    statuses, _ = roadmap_statuses(api)
    not_done = [n for n in ls if statuses.get(n) != "mastered"]
    if not_done:
        fail(f"not mastered: {not_done}")
    # Locked / coming-soon nodes refuse to start.
    for nid, want in (("a2-past-simple-regular", "node_coming_soon"), ("a1-self-introduction", "node_coming_soon")):
        status, body = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": nid})
        if status != 409:
            fail(f"{nid} should not start: {status} {body}")
    print(f"  ✓ all {len(ls)} ready lessons mastered by {EMAIL_A}")
    sign_in(api, EMAIL_B, register=False)
    _, d = call(api, "GET", "/api/v1/dashboard")
    if d["courses"] or d["openSessions"] or d["totalMastered"]:
        fail(f"{EMAIL_B} sees someone else's progress: {d}")
    print(f"  ✓ {EMAIL_B} sees none of it")


def verify(api, curriculum):
    sign_in(api, EMAIL_A, register=False)
    statuses, rm = roadmap_statuses(api)
    ls = lessons(curriculum)
    missing = [n for n in ls if statuses.get(n) != "mastered"]
    if missing:
        fail(f"progress lost after restart: {missing}")
    _, rm_id = call(api, "GET", f"/api/v1/courses/{COURSE}/roadmap", lang="id")
    first = rm_id["nodes"][0]
    if first["title"] != "Alfabet & Bunyi" or rm_id["course"]["title"] != "Bahasa Inggris":
        fail(f"Indonesian roadmap not localized: {first['title']!r} / {rm_id['course']['title']!r}")
    print(f"  ✓ Accept-Language id → {rm_id['course']['title']}: {first['title']}")
    _, d = call(api, "GET", "/api/v1/dashboard")
    if d["flashcards"]["total"] == 0:
        fail(f"flashcards lost after restart: {d['flashcards']}")
    print(f"  ✓ flashcards kept: {d['flashcards']['total']} saved, {d['flashcards']['due']} due")
    c = d["courses"][0]["progress"]
    print(f"  ✓ after restart and fresh login: {d['totalMastered']} mastered, level {c['currentLevel']}, "
          f"{c['percent']}% to {c['targetLevel']}, streak {d['streak']['current']} day(s)")


if __name__ == "__main__":
    mode, api, curriculum = sys.argv[1:4]
    {"learn": learn, "verify": verify}[mode](api, curriculum)
