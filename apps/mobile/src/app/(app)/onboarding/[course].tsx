import { useLocalSearchParams, useRouter } from 'expo-router';
import { useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { Chip, ErrorState, LoadingState, PageDots } from '@/components/ui/Bits';
import { Card } from '@/components/ui/Card';
import { DotPattern } from '@/components/ui/Decor';
import { Icon } from '@/components/ui/Icon';
import { IconButton, PillButton } from '@/components/ui/PillButton';
import { Eyebrow, Heading, Text } from '@/components/ui/Text';
import { TextField } from '@/components/ui/TextField';
import { useT } from '@/i18n';
import type { OnboardingOption, OnboardingQuestion, ProfileView } from '@/lib/api';
import { useErrorText } from '@/lib/errors';
import { useProfile, useSaveProfile } from '@/lib/queries';
import { colors, radius } from '@/theme/tokens';

const textKey = (qid: string) => `${qid}_text`;
const GOAL = 'goal';

function OptionCard({ option, selected, onPress }: { option: OnboardingOption; selected: boolean; onPress: () => void }) {
  const [hovered, setHovered] = useState(false);
  return (
    <Pressable
      onPress={onPress}
      onHoverIn={() => setHovered(true)}
      onHoverOut={() => setHovered(false)}
      accessibilityRole="radio"
      accessibilityState={{ selected }}
      accessibilityLabel={option.description ? `${option.label}, ${option.description}` : option.label}
      style={[
        styles.option,
        {
          borderColor: selected ? colors.accent : hovered ? colors.lineStrong : colors.line,
          backgroundColor: selected ? colors.accentSoft : colors.card,
        },
      ]}
    >
      <View style={{ flex: 1, gap: 2 }}>
        <Text weight="medium" tight color={colors.ink}>
          {option.label}
        </Text>
        {option.description ? (
          <Text size={13} color={colors.muted}>
            {option.description}
          </Text>
        ) : null}
      </View>
      <View style={[styles.radio, selected && { backgroundColor: colors.accent, borderColor: colors.accent }]}>
        {selected ? <Icon name="check" size={12} color={colors.white} strokeWidth={2.5} /> : null}
      </View>
    </Pressable>
  );
}

// Answer a course's onboarding questions one per screen, then review the
// plan (and target level) before saving. Editing starts from saved answers.
function Wizard({ course, profile }: { course: string; profile: ProfileView }) {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const t = useT();
  const errorText = useErrorText();
  const save = useSaveProfile();
  const questions = profile.questions;
  const [step, setStep] = useState(0);
  const [answers, setAnswers] = useState<Record<string, string>>(profile.answers);
  const [target, setTarget] = useState(profile.targetLevel);
  const total = questions.length;
  const onSummary = step >= total;
  const q: OnboardingQuestion | undefined = questions[step];
  const back = () => (router.canGoBack() ? router.back() : router.replace(`/courses/${course}`));

  const choose = (question: OnboardingQuestion, option: OnboardingOption) => {
    setAnswers((a) => {
      const next = { ...a, [question.id]: option.id };
      if (!option.freeText) delete next[textKey(question.id)];
      return next;
    });
    // The goal suggests a target level; the learner can still change it.
    if (question.id === GOAL && option.targetLevel) setTarget(option.targetLevel);
  };

  const submit = (input: Parameters<typeof save.mutate>[0]['input']) =>
    save.mutate({ course, input }, { onSuccess: () => router.replace(`/courses/${course}`) });

  const labelFor = (question: OnboardingQuestion) => {
    const o = question.options.find((x) => x.id === answers[question.id]);
    if (!o) return t('onboarding.notAnswered');
    return o.freeText && answers[textKey(question.id)] ? answers[textKey(question.id)] : o.label;
  };

  const header = (
    <View style={[styles.header, { paddingTop: insets.top + 12 }]}>
      <View style={styles.headerRow}>
        <IconButton icon="close" label={t('onboarding.close')} variant="soft" onPress={back} />
        <View style={{ flex: 1 }}>
          <Eyebrow>{t('onboarding.eyebrow', { course: profile.course.title })}</Eyebrow>
          <Text weight="medium" tight size={16} numberOfLines={1}>
            {onSummary ? t('onboarding.summaryTitle') : t('onboarding.stepOf', { n: step + 1, total })}
          </Text>
        </View>
        <PageDots count={total + 1} active={step} />
      </View>
    </View>
  );

  const selected = q ? q.options.find((o) => o.id === answers[q.id]) : undefined;
  const body = q ? (
    <View style={{ gap: 20 }}>
      <View style={{ gap: 8 }}>
        <Heading size={30}>{q.title}</Heading>
        {q.help ? <Text color={colors.muted}>{q.help}</Text> : null}
      </View>
      <View style={{ gap: 10 }} accessibilityRole="radiogroup">
        {q.options.map((o) => (
          <OptionCard key={o.id} option={o} selected={answers[q.id] === o.id} onPress={() => choose(q, o)} />
        ))}
      </View>
      {selected?.freeText ? (
        <TextField
          label={t('onboarding.otherLabel')}
          placeholder={t('onboarding.otherPlaceholder')}
          value={answers[textKey(q.id)] ?? ''}
          onChangeText={(v) => setAnswers((a) => ({ ...a, [textKey(q.id)]: v }))}
          maxLength={80}
          autoFocus
        />
      ) : null}
    </View>
  ) : (
    <View style={{ gap: 20 }}>
      <View style={{ gap: 8 }}>
        <Eyebrow color={colors.emeraldDeep}>{t('onboarding.summaryEyebrow')}</Eyebrow>
        <Heading size={30}>{t('onboarding.summaryTitle')}</Heading>
        <Text color={colors.muted}>{t('onboarding.summaryText')}</Text>
      </View>
      <Card padding={0} spotlight={false}>
        {questions.map((question, i) => (
          <Pressable
            key={question.id}
            onPress={() => setStep(i)}
            accessibilityRole="button"
            accessibilityLabel={`${question.title}: ${labelFor(question)}. ${t('onboarding.edit')}`}
            style={[styles.summaryRow, i > 0 && { borderTopWidth: 1, borderTopColor: colors.line }]}
          >
            <View style={{ flex: 1, gap: 2 }}>
              <Text size={13} color={colors.muted}>
                {question.title}
              </Text>
              <Text weight="medium" tight color={answers[question.id] ? colors.ink : colors.muted}>
                {labelFor(question)}
              </Text>
            </View>
            <Text size={13} color={colors.accent}>
              {t('onboarding.edit')}
            </Text>
          </Pressable>
        ))}
      </Card>
      <View style={{ gap: 10 }}>
        <Eyebrow>{t('onboarding.targetLabel')}</Eyebrow>
        <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
          {profile.levels.map((l) => (
            <Chip key={l} label={l} active={l === target} onPress={() => setTarget(l)} />
          ))}
        </View>
        <Text size={13} color={colors.muted}>
          {t('onboarding.targetHelp')}
        </Text>
      </View>
      {save.isError ? (
        <Text size={14} color={colors.danger}>
          {errorText(save.error)}
        </Text>
      ) : null}
    </View>
  );

  const answered = q ? !!answers[q.id] : true;
  const footer = (
    <View style={[styles.footer, { paddingBottom: Math.max(insets.bottom, 12) + 4 }]}>
      <View style={styles.footerRow}>
        {step > 0 ? (
          <PillButton label={t('onboarding.back')} variant="secondary" onPress={() => setStep(step - 1)} />
        ) : !profile.completed ? (
          // Skipping records the choice so the course page stops offering it.
          <Pressable onPress={() => submit({ skip: true })} disabled={save.isPending} accessibilityRole="button" style={styles.textButton}>
            <Text size={14} color={colors.muted}>
              {t('onboarding.skipAll')}
            </Text>
          </Pressable>
        ) : null}
        <View style={{ flex: 1 }} />
        {onSummary ? (
          <PillButton
            label={profile.completed ? t('onboarding.save') : t('onboarding.start')}
            icon="arrowUpRight"
            loading={save.isPending}
            onPress={() => submit({ answers, targetLevel: target })}
          />
        ) : (
          <PillButton
            label={answered || q?.required ? t('onboarding.next') : t('onboarding.skipQuestion')}
            icon={answered || q?.required ? 'arrowRight' : undefined}
            variant={answered || q?.required ? 'primary' : 'secondary'}
            disabled={q?.required && !answered}
            onPress={() => setStep(step + 1)}
          />
        )}
      </View>
    </View>
  );

  return (
    <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <View style={styles.column}>
        {header}
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          {body}
        </ScrollView>
        {footer}
      </View>
    </KeyboardAvoidingView>
  );
}

export default function OnboardingScreen() {
  const { course } = useLocalSearchParams<{ course: string }>();
  const { data, isPending, isError, error, refetch } = useProfile(course);
  const t = useT();
  const errorText = useErrorText();
  return (
    <View style={styles.root}>
      <DotPattern />
      {isPending ? (
        <View style={styles.center}>
          <LoadingState label={t('onboarding.loading')} />
        </View>
      ) : isError ? (
        <View style={styles.center}>
          <ErrorState title={t('onboarding.loadError')} message={errorText(error)} onRetry={refetch} />
        </View>
      ) : (
        // Answers live in the wizard's state, so switching language only
        // swaps the question text.
        <Wizard course={course} profile={data} />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: colors.canvas },
  column: { flex: 1, width: '100%', maxWidth: 640, alignSelf: 'center' },
  header: { paddingHorizontal: 16, paddingBottom: 8 },
  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    backgroundColor: 'rgba(255,255,255,0.8)',
    borderWidth: 1,
    borderColor: colors.line,
    borderRadius: 24,
    padding: 8,
    paddingRight: 16,
  },
  scroll: { paddingHorizontal: 16, paddingVertical: 16 },
  option: { flexDirection: 'row', alignItems: 'center', gap: 12, borderWidth: 1.5, borderRadius: radius.xxl + 4, paddingHorizontal: 16, paddingVertical: 14, minHeight: 56 },
  radio: { width: 22, height: 22, borderRadius: 11, borderWidth: 1.5, borderColor: colors.lineStrong, alignItems: 'center', justifyContent: 'center' },
  summaryRow: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingHorizontal: 20, paddingVertical: 14 },
  footer: { paddingHorizontal: 16, paddingTop: 8 },
  footerRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  textButton: { paddingHorizontal: 12, paddingVertical: 10 },
  center: { flex: 1, justifyContent: 'center', padding: 16 },
});
