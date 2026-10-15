"""HTTP driver for scripts/e2e.sh. Uses only the standard library.

  e2e_driver.py learn  <api> <course_dir>   sign up, enrol, walk every lesson's steps and master it
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


def step_call(api, sid, n, action, body=None):
    return call(api, "POST", f"/api/v1/sessions/{sid}/steps/{n}/{action}", body)


def answer_for(q):
    # The driver can't see the answer (the server keeps it until answered), so it guesses.
    return {"choice": 0} if q["type"] == "mcq" else {"tokens": q["tokens"]}


def walk_steps(api, node_id, s, helper):
    """Goes through every step; on the helper lesson it also asks, uses both hints,
    gives up, takes the similar question and checks the rules on the way."""
    sid = s["id"]
    tried = False
    while not s["lesson"]["checkpoint"]:
        n = s["lesson"]["stepIndex"]
        step = s["lesson"]["steps"][n]
        q = step["question"]
        if helper and n == 0 and not step["asks"]:
            status, s = step_call(api, sid, 0, "ask", {"content": "Can you explain this again?"})
            if status != 200 or len(s["lesson"]["steps"][0]["asks"]) != 1:
                fail(f"{node_id}: ask in step: {status} {s}")
            continue
        if q and not (q["answered"] or q["revealed"]):
            if q["answer"] is not None or q["solution"] or q["explanation"]:
                fail(f"{node_id}: the answer leaked before answering: {q}")
            status, body = step_call(api, sid, n, "next")
            if status != 409 or body["error"]["code"] != "answer_first":
                fail(f"{node_id}: moving on before answering: {status} {body}")
            if helper and not tried:
                tried = True
                for _ in range(2):
                    status, s = step_call(api, sid, n, "hint")
                    if status != 200:
                        fail(f"{node_id}: hint: {status} {s}")
                status, body = step_call(api, sid, n, "hint")
                if status != 409 or body["error"]["code"] != "no_hints_left":
                    fail(f"{node_id}: a third hint: {status} {body}")
                status, s = step_call(api, sid, n, "reveal")
                q = s["lesson"]["steps"][n]["question"]
                if status != 200 or not q["revealed"] or not q["explanation"]:
                    fail(f"{node_id}: give up: {status} {q}")
                status, s = step_call(api, sid, n, "similar")
                sim = s["lesson"]["steps"][n]["similar"] if status == 200 else None
                if not sim or sim["answer"] is not None:
                    fail(f"{node_id}: similar question: {status} {s}")
                status, s = step_call(api, sid, n, "answer", {**answer_for(sim), "similar": True})
                if status != 200 or not s["lesson"]["steps"][n]["similar"]["answered"]:
                    fail(f"{node_id}: answer similar: {status} {s}")
                status, body = step_call(api, sid, n, "similar")
                if status != 409 or body["error"]["code"] != "similar_used":
                    fail(f"{node_id}: second similar question: {status} {body}")
            else:
                status, s = step_call(api, sid, n, "answer", answer_for(q))
                if status != 200:
                    fail(f"{node_id}: answer step {n}: {status} {s}")
            continue
        if helper and step["kind"] == "artifact" and step["artifact"]:
            status, note = call(api, "POST", "/api/v1/notes", {"course": COURSE, "nodeId": node_id, "artifact": step["artifact"]})
            if status != 200:
                fail(f"{node_id}: save note: {status} {note}")
        status, s = step_call(api, sid, n, "next")
        if status != 200:
            fail(f"{node_id}: next from step {n}: {status} {s}")
    return s


def drive(api, node_id, lesson, helper=False):
    status, s = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": node_id})
    if status != 200:
        fail(f"start {node_id}: {status} {s}")
    if s["phase"] != "LEARNING" or not s["lesson"] or len(s["lesson"]["steps"]) != 1 or s["checkpoint"] or not s["lesson"]["greeting"]:
        fail(f"{node_id}: lesson should open on its first step with the checkpoint ahead: {s.get('phase')}")
    status, body = call(api, "POST", f"/api/v1/sessions/{s['id']}/checkpoint")
    if status != 409 or body["error"]["code"] != "finish_steps_first":
        fail(f"{node_id}: checkpoint before the steps: {status} {body}")
    if helper:
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
    total = s["lesson"]["totalSteps"]
    s = walk_steps(api, node_id, s, helper)
    if helper:
        print(f"  ✓ {node_id}: {total} steps — asked, 2 hints then gave up, similar question once; flashcards saved and reviewed")

    # A failing attempt first: must not unlock anything, and the lesson stays open.
    cp = start_checkpoint(api, node_id, s, lesson, attempt=1)
    wrong = [{"itemId": it["id"], "answer": "zzz"} for it in cp["items"]]
    status, r = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"checkpointId": cp["id"], "answers": wrong})
    if status != 200 or r["passed"] or r["unlocked"]:
        fail(f"{node_id}: failing attempt misbehaved: {status} {r}")
    # The next attempt's questions may already be prepared in the background; the failed one is closed.
    nxt = r["session"]["checkpoint"]
    if r["session"]["phase"] != "LEARNING" or not r["session"]["canChat"] or (nxt and nxt["id"] == cp["id"]):
        fail(f"{node_id}: expected the lesson to stay open after failing, got {r['session']['phase']}")
    review = r["attempt"]["review"]["items"]
    if len(review) != len(cp["items"]) or any(not it["explanation"] or it["verdict"] == "correct" for it in review):
        fail(f"{node_id}: every answer should be explained: {review}")
    s = r["session"]
    # A retake is a new checkpoint.
    cp = start_checkpoint(api, node_id, s, lesson, attempt=2)
    right = [{"itemId": it["id"], "answer": lesson["assessment"][i]["acceptableAnswers"][0]} for i, it in enumerate(cp["items"])]
    status, r = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"checkpointId": cp["id"], "answers": right})
    if status != 200 or not r["passed"]:
        fail(f"{node_id}: passing attempt failed: {status} {r}")
    history = r["session"]["attempts"]
    if [a["passed"] for a in history] != [True, False]:
        fail(f"{node_id}: checkpoint history should hold both attempts, newest first: {history}")
    summary = r["session"]["lesson"]["summary"]
    if not summary or not summary["mastered"]:
        fail(f"{node_id}: lesson summary missing: {summary}")
    if helper and summary["flashcardsAdded"] < 1:
        fail(f"{node_id}: the given-up question should become a mistake flashcard: {summary}")
    # Repeated completion request is idempotent.
    status, again = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"answers": right})
    if status != 200 or not again["alreadyCompleted"] or again["newlyMastered"]:
        fail(f"{node_id}: resubmission not idempotent: {status} {again}")
    # Opening the lesson again shows the finished lesson, not a new one.
    status, back = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": node_id})
    if status != 200 or back["id"] != s["id"] or back["phase"] != "MASTERED" or len(back["attempts"]) != 2:
        fail(f"{node_id}: a mastered lesson should reopen where it was finished: {status} {back.get('id')} {back.get('phase')}")
    if helper:
        # Practice after mastery: recorded, progress untouched.
        cp = start_checkpoint(api, node_id, back, lesson, attempt=3, practice=True)
        status, p = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"checkpointId": cp["id"], "answers": wrong})
        if status != 200 or p["passed"] or not p["attempt"]["practice"] or p["session"]["node"]["status"] != "mastered":
            fail(f"{node_id}: a practice checkpoint must not change progress: {status} {p}")
        print(f"  ✓ {node_id}: practice checkpoint after mastery recorded ({p['attempt']['score']}/100), still mastered")
    unlocked = [n["id"] for n in r["unlocked"]]
    extra = f", {summary['flashcardsAdded']} mistake card(s)" if summary["flashcardsAdded"] else ""
    print(f"  ✓ {node_id}: {r['attempt']['score']}/100, unlocked {unlocked or '—'}{extra}")


def start_checkpoint(api, node_id, s, lesson, attempt, practice=False):
    status, cp = call(api, "POST", f"/api/v1/sessions/{s['id']}/checkpoint")
    if status != 200 or cp["attempt"] != attempt or cp["practice"] != practice or len(cp["items"]) != len(lesson["assessment"]):
        fail(f"{node_id}: checkpoint {attempt}: {status} {cp}")
    if any("acceptedAnswers" in it or "rubric" in it for it in cp["items"]):
        fail(f"{node_id}: the checkpoint must not carry its answers")
    return cp


def enroll(api):
    # Personalisation is off for now: no onboarding questions, the course page enrols directly.
    status, rm = call(api, "GET", f"/api/v1/courses/{COURSE}/roadmap")
    if status != 200 or rm["profile"]["available"]:
        fail(f"onboarding should be off: {status} {rm.get('profile')}")
    status, rm = call(api, "POST", f"/api/v1/courses/{COURSE}/enroll")
    if status != 200 or not rm["enrolled"] or rm["enrollment"]["targetLevel"] != "B2":
        fail(f"enroll: {status} {rm}")
    if any(n["recommended"] for n in rm["nodes"]):
        fail("no goal badges while personalisation is off")
    print("  ✓ enrolled without onboarding (personalisation off), target B2")


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
    enroll(api)
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
    for nid, want in (("c1-inversion", "node_coming_soon"), ("c2-stylistic-nuance", "node_coming_soon")):
        status, body = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": nid})
        if status != 409:
            fail(f"{nid} should not start: {status} {body}")
    print(f"  ✓ all {len(ls)} ready lessons mastered by {EMAIL_A}")
    for ck in ("a0-checkpoint", "a1-checkpoint", "a2-checkpoint", "b1-checkpoint"):
        level_checkpoint(api, ls, ck)
    sign_in(api, EMAIL_B, register=False)
    _, d = call(api, "GET", "/api/v1/dashboard")
    if d["courses"] or d["openSessions"] or d["totalMastered"]:
        fail(f"{EMAIL_B} sees someone else's progress: {d}")
    print(f"  ✓ {EMAIL_B} sees none of it")


def level_checkpoint(api, ls, ck):
    # A level checkpoint: no steps, questions spread over the level's lessons.
    status, s = call(api, "POST", "/api/v1/sessions", {"course": COURSE, "nodeId": ck})
    if status != 200 or s["lesson"] or not s["level"] or any(l["status"] != "mastered" for l in s["level"]["lessons"]):
        fail(f"{ck} should open with its mastered lessons: {status} {s.get('level')}")
    status, cp = call(api, "POST", f"/api/v1/sessions/{s['id']}/checkpoint")
    if status != 200 or not 10 <= len(cp["items"]) <= 15 or not all(it.get("lessonId") for it in cp["items"]):
        fail(f"{ck}: {status} {cp}")
    # The offline provider uses hand-written questions from the level's lessons.
    def key(it):
        for a in ls[it["lessonId"]]["assessment"]:
            if a["prompt"] == it["prompt"]:
                return a["acceptableAnswers"][0]
        fail(f"no hand-written question matches {it['prompt']!r}")
    right = [{"itemId": it["id"], "answer": key(it)} for it in cp["items"]]
    status, r = call(api, "POST", f"/api/v1/sessions/{s['id']}/assessment", {"checkpointId": cp["id"], "answers": right})
    lessons_scored = r.get("attempt", {}).get("review", {}).get("lessons", {}) if status == 200 else {}
    asked = {it["lessonId"] for it in cp["items"]}
    if status != 200 or not r["passed"] or set(lessons_scored) != asked or r["session"]["node"]["status"] != "mastered":
        fail(f"{ck} should pass: {status} {r}")
    print(f"  ✓ {ck}: {len(cp['items'])} questions from {len(asked)} lessons, {r['attempt']['score']}/100, "
          f"level {s['node']['level']} complete")


def verify(api, curriculum):
    sign_in(api, EMAIL_A, register=False)
    statuses, rm = roadmap_statuses(api)
    ls = lessons(curriculum)
    missing = [n for n in ls if statuses.get(n) != "mastered"]
    if missing:
        fail(f"progress lost after restart: {missing}")
    _, rm_id = call(api, "GET", f"/api/v1/courses/{COURSE}/roadmap", lang="id")
    first = next(n for n in rm_id["nodes"] if n["id"] == "a0-alphabet-sounds")
    if first["title"] != "Alfabet & Bunyi" or rm_id["course"]["title"] != "Bahasa Inggris":
        fail(f"Indonesian roadmap not localized: {first['title']!r} / {rm_id['course']['title']!r}")
    print(f"  ✓ Accept-Language id → {rm_id['course']['title']}: {first['title']}")
    _, d = call(api, "GET", "/api/v1/dashboard")
    if d["flashcards"]["total"] == 0:
        fail(f"flashcards lost after restart: {d['flashcards']}")
    print(f"  ✓ flashcards kept: {d['flashcards']['total']} saved, {d['flashcards']['due']} due")
    _, notes = call(api, "GET", "/api/v1/notes")
    if not notes["notes"]:
        fail("saved notes lost after restart")
    if d["weekly"]["steps"] == 0:
        fail(f"lesson steps should count as activity: {d['weekly']}")
    print(f"  ✓ {len(notes['notes'])} note(s) kept; {d['weekly']['steps']} lesson steps this week")
    c = d["courses"][0]["progress"]
    print(f"  ✓ after restart and fresh login: {d['totalMastered']} mastered, level {c['currentLevel']}, "
          f"{c['percent']}% to {c['targetLevel']}, streak {d['streak']['current']} day(s)")


if __name__ == "__main__":
    mode, api, curriculum = sys.argv[1:4]
    {"learn": learn, "verify": verify}[mode](api, curriculum)
