#!/usr/bin/env python3
"""Inventory every authored assessment and lesson-step question for review.

Candidates are prompts for human review, not proof of a pedagogical defect.
Run: python3 scripts/checkpoint_audit.py --output review/checkpoint-audit-candidates.json
"""
import argparse
import collections
import json
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parents[1] / 'packages/courses/english'
LEVELS = ['A0', 'A1', 'A2', 'B1', 'B2', 'C1', 'C2']
WORDS = re.compile(r"[a-zA-Z]+(?:'[a-zA-Z]+)?")
FORMATTING = re.compile(r'(?i)dash|hyphen|strip|comma|capital|huruf besar|huruf kapital|koma')
LIST = re.compile(r'(?i)(?:write|list|name|tulis|sebutkan).*(?:vowels|vowel letters|words.*capital|huruf.*vokal|kata.*kapital)')
PRODUCTION = re.compile(r'(?i)translate|rewrite|correct the|make (?:a|it)|write.*sentence|terjemah|tulis ulang|perbaiki|susun.*kalimat')
SOUND_TOPIC = re.compile(r'(?i)\b(?:bunyi|vokal|pengucapan|diucapkan|sounds?|vowels?|pronunciation|pronounced|rhymes?|berima)\b')
SOUND_QUOTED_REFERENCE = re.compile(r"(?i)\b(?:seperti(?:\s+(?:pada|di|dalam))?|(?:as|like)(?:\s+in)?|rhymes?\s+with|berima\s+dengan)\s+(?:(?:the\s+word|word|kata)\s+)?['\"‘’“”*`]([a-z]+)['\"‘’“”*`](?:$|[^a-z])")
SOUND_PLAIN_REFERENCE = re.compile(r"(?i)\b(?:seperti(?:\s+(?:pada|di|dalam))?|as\s+in|(?:sounds?|vowels?)\s+(?:as|like)|like\s+in|rhymes?\s+with|berima\s+dengan)\s+(?:(?:the\s+word|word|kata)\s+)?([a-z]+)\b")


def strings(value):
    if isinstance(value, str):
        yield value
    elif isinstance(value, dict):
        for key, child in value.items():
            if key not in ('assessment', 'i18n', 'resources'):
                yield from strings(child)
    elif isinstance(value, list):
        for child in value:
            yield from strings(child)


def words(value):
    return {w.lower() for text in strings(value) for w in WORDS.findall(text)}


def audit():
    lessons = {p.stem: json.loads(p.read_text()) for p in sorted((ROOT / 'lessons').glob('*.json'))}
    plans = {p.stem: json.loads(p.read_text()) for p in sorted((ROOT / 'plans').glob('*.json'))}
    nodes = {n['id']: n for level in LEVELS for n in json.loads((ROOT / (level.lower()+'.json')).read_text())['nodes']}
    exposures = {node: words(l.get('material', [])) | words(l.get('keyPoints', [])) | words(l.get('intro', '')) | words(plans.get(node, {})) for node, l in lessons.items()}
    records = []
    totals = collections.Counter()
    def prerequisites(node, seen=None):
        seen = set() if seen is None else seen
        for parent in nodes[node].get('prerequisiteIds', []):
            if parent not in seen:
                seen.add(parent)
                prerequisites(parent, seen)
        return seen
    for node, lesson in lessons.items():
        level = nodes[node]['level']
        exposure = set(exposures[node])
        for parent in prerequisites(node):
            exposure |= exposures.get(parent, set())
        if level != 'A0':
            for parent, data in lessons.items():
                if LEVELS.index(nodes[parent]['level']) < LEVELS.index(level):
                    exposure |= exposures[parent]
        groups = [('assessment', lesson['assessment'])]
        translated_assessment = []
        translation = lesson.get('i18n', {}).get('id', {})
        for item in lesson['assessment']:
            q = dict(item)
            q['prompt'] = translation.get('assessment', {}).get(item['id'], item['prompt'])
            q['displayOptions'] = translation.get('assessmentOptions', {}).get(item['id'], [])
            translated_assessment.append(q)
        groups.append(('assessment.id', translated_assessment))
        groups += [(name, [dict(q, id=f'{name}-{i+1}') for i, q in enumerate(lesson.get(name, []))])
                   for name in ('guidedPractice', 'independentPractice')]
        groups += [(f'plan.{lang}', [dict(s['question'], id=f'step-{i+1}') for i, s in enumerate(plan['steps']) if 'question' in s]) for lang, plan in plans.get(node, {}).items()]
        for lang in ('en', 'id'):
            material_questions = []
            for section in lesson.get('material', []):
                for index, block in enumerate(section.get('blocks', [])):
                    if block.get('type') != 'check':
                        continue
                    q = dict(block, id=f"{section['id']}-block-{index+1}", type='mcq')
                    if lang == 'id':
                        q['prompt'] = block.get('i18n', {}).get('id', {}).get('prompt', block['prompt'])
                        q['displayOptions'] = block.get('i18n', {}).get('id', {}).get('options', [])
                    material_questions.append(q)
            groups.append((f'material.{lang}', material_questions))
        for source, questions in groups:
            for q in questions:
                kind = q.get('type', 'text')
                totals[f'{level}:{source}:{kind}'] += 1
                prompt = q.get('prompt', '')
                flags = []
                if FORMATTING.search(prompt): flags.append('format_or_capitalization_target_review')
                if kind == 'text' and LIST.search(prompt): flags.append('list_in_free_text')
                if level in ('A0', 'A1') and kind == 'text' and PRODUCTION.search(prompt): flags.append('beginner_production_review')
                if kind == 'text' and re.search(r'(?i)spell.*(?:dash|hyphen|letters)|eja.*(?:strip|huruf)', prompt): flags.append('spelling_in_free_text')
                targets = q.get('acceptableAnswers') or q.get('expected') or q.get('solution') or q.get('correctOption') or q.get('correctOptions')
                if not targets and q.get('pairs'):
                    targets = q['pairs']
                if not targets and kind == 'mcq' and 0 <= q.get('answer', -1) < len(q.get('options', [])):
                    targets = q['options'][q['answer']]
                targets = targets or []
                references = {m.group(1).lower() for pattern in (SOUND_QUOTED_REFERENCE, SOUND_PLAIN_REFERENCE)
                              for m in pattern.finditer(prompt)} if SOUND_TOPIC.search(prompt) else set()
                if any(' '.join(WORDS.findall(text.lower())) in references for text in strings(targets)):
                    flags.append('sound_comparison_reuses_reference_as_answer')
                # Proper names/numbers and distractors are not evidence of untaught target language.
                unseen = sorted(words(targets) - exposure - words(prompt)) if level in ('A0', 'A1') else []
                if unseen: flags.append('target_vocabulary_not_found_in_material_or_prerequisites')
                records.append({'lesson': node, 'level': level, 'source': source, 'id': q['id'], 'type': kind,
                                'prompt': prompt, 'canonicalOptions': q.get('options', []),
                                'visibleOptions': q.get('displayOptions') or q.get('options', []),
                                'hasTranslatedOptions': bool(q.get('displayOptions')),
                                'flags': flags, 'unseenTargetWords': unseen})
    return {'method': 'All assessment items, practice templates, EN/ID assessment prompts, material checks and step questions. Option labels include the translations learners actually see. Vocabulary exposure uses learner-facing material and plans, excluding teacher notes and assessment keys. Flags require contextual human review.',
            'totals': dict(sorted(totals.items())), 'questions': records}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=pathlib.Path)
    args = parser.parse_args()
    result = audit()
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    flagged = [q for q in result['questions'] if q['flags']]
    print(f"Inventoried {len(result['questions'])} questions; {len(flagged)} review candidates.")
    for q in flagged:
        print(f"{q['lesson']} {q['source']} {q['id']}: {', '.join(q['flags'])}" + (f" ({', '.join(q['unseenTargetWords'])})" if q['unseenTargetWords'] else ''))


if __name__ == '__main__':
    main()
