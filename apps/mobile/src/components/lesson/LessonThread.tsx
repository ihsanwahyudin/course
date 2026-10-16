import { useState } from 'react';
import { View } from 'react-native';

import { PillButton } from '@/components/ui/PillButton';
import { Text } from '@/components/ui/Text';
import { useT } from '@/i18n';
import type { AnswerInput, Artifact, Flashcard, QuestionView, Session, StepView } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useSaveFlashcards, useStepAction, type StepAction } from '@/lib/queries';
import { colors } from '@/theme/tokens';
import { AskBox } from './AskBox';
import { FlashcardDeck } from './FlashcardDeck';
import { QuestionCard, type QuestionBusy } from './QuestionCard';
import { ActionRow, DoneStep, Indented, MemoryChip, StepLabel, TutorLine, UpcomingStep } from './StepParts';
import { ArtifactPreview, ConceptCard, ExamplesCard, ResourceList, TimelineCard, WidgetCard } from './Widgets';

type Ctx = {
  session: Session;
  openArtifact: (a: Artifact) => void;
};

function Cards({ ctx, cards }: { ctx: Ctx; cards: Flashcard[] }) {
  const save = useSaveFlashcards();
  const [saved, setSaved] = useState(false);
  return (
    <FlashcardDeck
      cards={cards}
      saving={save.isPending}
      saved={saved}
      onSave={() => save.mutate({ course: ctx.session.courseSlug, nodeId: ctx.session.nodeId, cards }, { onSuccess: () => setSaved(true) })}
    />
  );
}

// The step's widget (everything except a question, which needs actions).
function Widget({ step, ctx }: { step: StepView; ctx: Ctx }) {
  switch (step.kind) {
    case 'concept':
      return <ConceptCard step={step} />;
    case 'examples':
      return <ExamplesCard step={step} />;
    case 'timeline':
      return step.timeline ? <TimelineCard step={step} timeline={step.timeline} /> : null;
    case 'flashcards':
      return step.flashcards?.length ? (
        <WidgetCard eyebrow={step.title}>
          <Cards ctx={ctx} cards={step.flashcards} />
        </WidgetCard>
      ) : null;
    case 'artifact':
      return step.artifact ? <ArtifactPreview artifact={step.artifact} onOpen={() => ctx.openArtifact(step.artifact!)} /> : null;
    case 'resources':
      return <ResourceList step={step} />;
    default:
      return null;
  }
}

const noop = () => {};

// Chat can be resumed on any opened step while the lesson session is open.
function StepChat({ step, ctx }: { step: StepView; ctx: Ctx }) {
  const act = useStepAction(ctx.session.id);
  const errorText = useErrorText();
  return (
    <AskBox
      asks={step.asks}
      pending={act.isPending && act.variables?.action === 'ask' ? act.variables.content : null}
      error={act.error && !act.isPending ? errorText(act.error) : undefined}
      onAsk={ctx.session.endedAt ? undefined : (content) => act.mutate({ action: 'ask', step: step.index, content })}
    />
  );
}

// A finished step: practice stays read-only; its chat can be continued.
function PastStep({ step, ctx }: { step: StepView; ctx: Ctx }) {
  const t = useT();
  return (
    <DoneStep n={step.index + 1} kind={step.kind} title={step.question?.prompt ?? step.title}>
      {step.text ? <TutorLine>{step.text}</TutorLine> : null}
      {step.question ? (
        <QuestionCard
          q={step.question}
          eyebrow={t('step.practice')}
          listenPath={`/api/v1/sessions/${ctx.session.id}/steps/${step.index}/listen`}
          busy={null}
          readOnly
          onAnswer={noop}
          onHint={noop}
          onReveal={noop}
        />
      ) : (
        <Widget step={step} ctx={ctx} />
      )}
      {step.similar ? <QuestionCard q={step.similar} eyebrow={t('step.similarLabel', { n: step.index + 1 })} busy={null} readOnly onAnswer={noop} onHint={noop} onReveal={noop} /> : null}
      {!step.question || step.asks.length > 0 ? <StepChat step={step} ctx={ctx} /> : null}
    </DoneStep>
  );
}

const missed = (q: QuestionView | null) => !!q && ((q.answered && !q.correct) || q.revealed);
const settled = (q: QuestionView | null) => !!q && (q.answered || q.revealed);

// The current step, shown in full with its actions.
function ActiveStep({ step, total, ctx, greeting }: { step: StepView; total: number; ctx: Ctx; greeting?: string }) {
  const t = useT();
  const errorText = useErrorText();
  const act = useStepAction(ctx.session.id);
  const n = step.index;
  const v = act.isPending ? act.variables : undefined;

  const run = (a: StepAction) => act.mutate(a);
  const busyFor = (similar: boolean): QuestionBusy => {
    if (!v || v.step !== n) return null;
    if (v.action === 'answer') return !!v.input.similar === similar ? 'answer' : null;
    if (v.action === 'hint' || v.action === 'reveal') return !!v.similar === similar ? v.action : null;
    return null;
  };
  const question = (q: QuestionView, similar: boolean, eyebrow: string) => (
    <QuestionCard
      key={`${n}-${similar}`}
      q={q}
      eyebrow={eyebrow}
      listenPath={`/api/v1/sessions/${ctx.session.id}/steps/${n}/listen`}
      busy={busyFor(similar)}
      onAnswer={(input: AnswerInput) => run({ action: 'answer', step: n, input: { ...input, similar } })}
      onHint={() => run({ action: 'hint', step: n, similar })}
      onReveal={() => run({ action: 'reveal', step: n, similar })}
    />
  );
  const next = (
    <PillButton key="next" label={t('step.continue')} icon="arrowRight" loading={v?.action === 'next'} disabled={act.isPending} onPress={() => run({ action: 'next', step: n })} />
  );
  const err = act.error && v === undefined && act.variables?.action !== 'ask' ? errorText(act.error) : null;

  let body;
  if (step.question) {
    const q = step.question;
    let after = null;
    if (settled(q)) {
      if (!missed(q) || q.hasAudio) {
        // Listening questions are recorded with the lesson, so there's no similar one.
        after = <ActionRow>{next}</ActionRow>;
      } else if (!step.similar) {
        after = (
          <View style={{ gap: 12 }}>
            <Text size={16}>{t('step.similarAsk')}</Text>
            <ActionRow>
              <PillButton
                label={t('step.trySimilar')}
                icon="undo"
                variant="secondary"
                loading={v?.action === 'similar'}
                disabled={act.isPending}
                onPress={() => run({ action: 'similar', step: n })}
              />
              {next}
            </ActionRow>
          </View>
        );
      } else {
        after = (
          <View style={{ gap: 14 }}>
            <StepLabel n={n + 1} total={total} kind="question" suffix={t('step.similarLabel', { n: n + 1 })} />
            {question(step.similar, true, t('step.practice'))}
            {settled(step.similar) ? (
              <>
                {missed(step.similar) ? <Text size={15} color={colors.muted}>{t('step.similarDone')}</Text> : null}
                <ActionRow>{next}</ActionRow>
              </>
            ) : null}
          </View>
        );
      }
    }
    body = (
      <>
        <Indented>{question(q, false, t('step.practice'))}</Indented>
        {after ? <Indented>{after}</Indented> : null}
      </>
    );
  } else {
    body = (
      <Indented>
        <Widget step={step} ctx={ctx} />
        <StepChat step={step} ctx={ctx} />
        <ActionRow>
          {next}
        </ActionRow>
      </Indented>
    );
  }

  return (
    <View style={{ gap: 14 }}>
      <StepLabel n={n + 1} total={total} kind={step.kind} />
      {greeting ? <TutorLine>{greeting}</TutorLine> : null}
      {step.text ? <TutorLine>{step.text}</TutorLine> : null}
      {body}
      {err ? (
        <Indented>
          <Text size={14} color={colors.danger} accessibilityLiveRegion="polite">
            {err}
          </Text>
        </Indented>
      ) : null}
    </View>
  );
}

// The lesson so far: finished steps collapsed, the current one in full, and
// a placeholder for what comes next.
export function LessonThread({ session, openArtifact }: { session: Session; openArtifact: (a: Artifact) => void }) {
  const lesson = session.lesson!;
  const ctx = { session, openArtifact };
  const current = lesson.checkpoint ? null : lesson.steps[lesson.stepIndex];
  return (
    <View style={{ gap: 14 }}>
      {lesson.memoryChip ? <MemoryChip chip={lesson.memoryChip} /> : null}
      {lesson.steps
        .filter((s) => s.done)
        .map((s) => (
          <PastStep key={s.index} step={s} ctx={ctx} />
        ))}
      {current ? (
        <View style={{ marginTop: 10, gap: 18 }}>
          <ActiveStep key={current.index} step={current} total={lesson.totalSteps} ctx={ctx} greeting={current.index === 0 ? lesson.greeting : undefined} />
          {current.index + 1 < lesson.totalSteps ? <UpcomingStep n={current.index + 2} /> : null}
        </View>
      ) : null}
    </View>
  );
}
