import { View } from 'react-native';

import { ComingSoonCard, CourseCard } from '@/components/CourseCard';
import { Headline } from '@/components/Headline';
import { PageHeader } from '@/components/PageHeader';
import { ErrorState, LoadingState } from '@/components/ui/Bits';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { useT } from '@/i18n';
import { useErrorText } from '@/lib/errors';
import { useCourses } from '@/lib/queries';

export default function CoursesScreen() {
  const courses = useCourses();
  const { wide, gap, contentWidth } = useLayout();
  const colW = wide ? span(contentWidth, gap, 3, 1) : contentWidth;
  const t = useT();
  const errorText = useErrorText();

  return (
    <Screen onRefresh={courses.refetch} refreshing={courses.isRefetching}>
      <PageHeader
        label={t('nav.courses')}
        eyebrow={courses.data ? (courses.data.length === 1 ? t('catalog.countOne') : t('catalog.count', { n: courses.data.length })) : t('nav.courses')}
        title={wide ? <Headline size={60} before={t('catalog.before')} gradient={t('catalog.gradient')} after={t('catalog.after')} /> : t('nav.courses')}
        description={t('catalog.description')}
      />
      {courses.isPending ? (
        <LoadingState label={t('catalog.loading')} />
      ) : courses.isError ? (
        <ErrorState title={t('landing.loadCoursesError')} message={errorText(courses.error)} onRetry={courses.refetch} />
      ) : (
        <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap, alignItems: 'stretch' }}>
          {courses.data.map((c, i) => (
            <CourseCard key={c.slug} course={c} reveal={i} style={{ width: colW }} />
          ))}
          <ComingSoonCard style={{ width: colW }} />
        </View>
      )}
    </Screen>
  );
}
