import { View } from 'react-native';

import { ComingSoonCard, CourseCard } from '@/components/CourseCard';
import { Headline } from '@/components/Headline';
import { PageHeader } from '@/components/PageHeader';
import { ErrorState, LoadingState } from '@/components/ui/Bits';
import { Screen, span, useLayout } from '@/components/ui/Screen';
import { useCourses } from '@/lib/queries';

export default function CoursesScreen() {
  const courses = useCourses();
  const { wide, gap, contentWidth } = useLayout();
  const colW = wide ? span(contentWidth, gap, 3, 1) : contentWidth;

  return (
    <Screen onRefresh={courses.refetch} refreshing={courses.isRefetching}>
      <PageHeader
        label="Courses"
        eyebrow={courses.data ? `${courses.data.length} course${courses.data.length === 1 ? '' : 's'}` : 'Courses'}
        title={wide ? <Headline size={60} before="Pick a" gradient="roadmap" after=" to follow." /> : 'Courses'}
        description="Each course is a roadmap of skills with lessons, an AI tutor and assessments. English is the first; more are coming."
      />
      {courses.isPending ? (
        <LoadingState label="Loading courses" />
      ) : courses.isError ? (
        <ErrorState title="Can't load courses" message={courses.error.message} onRetry={courses.refetch} />
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
