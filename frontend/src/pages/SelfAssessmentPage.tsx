import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Paper,
  Title,
  Text,
  Button,
  Progress,
  Tabs,
  Radio,
  Group,
  Textarea,
  Card,
  Badge,
  Stack,
  Alert,
  ActionIcon,
  Loader,
  Center,
} from '@mantine/core';
import {
  IconChevronLeft,
  IconChevronRight,
  IconCheck,
  IconAlertCircle,
  IconArrowLeft,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { selfAssessmentService } from '../services/selfAssessment';
import adminService from '../services/admin';
import type {
  SelfAssessment,
  CatalogWithDetails,
  AssessmentResponseWithDetails,
  AssessmentCompleteness,
  CategoryWithPaths,
  Level,
} from '../types';

export default function SelfAssessmentPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const assessmentId = parseInt(id || '0');

  const [assessment, setAssessment] = useState<SelfAssessment | null>(null);
  const [catalog, setCatalog] = useState<CatalogWithDetails | null>(null);
  const [responses, setResponses] = useState<AssessmentResponseWithDetails[]>([]);
  const [completeness, setCompleteness] = useState<AssessmentCompleteness | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [selectedPath, setSelectedPath] = useState<number | null>(null);
  const [selectedLevel, setSelectedLevel] = useState<number | null>(null);
  const [justification, setJustification] = useState('');
  const [levelViewStart, setLevelViewStart] = useState(0);

  const LEVELS_PER_VIEW = 3;

  useEffect(() => {
    loadData();
  }, [assessmentId]);

  useEffect(() => {
    if (catalog?.categories && catalog.categories.length > 0 && !activeCategory) {
      setActiveCategory(catalog.categories[0].id.toString());
    }
  }, [catalog]);

  useEffect(() => {
    // Load existing response for active category
    if (activeCategory && responses.length > 0) {
      const categoryId = parseInt(activeCategory);
      const existingResponse = responses.find(r => r.category_id === categoryId);
      
      if (existingResponse) {
        setSelectedPath(existingResponse.path_id);
        setSelectedLevel(existingResponse.level_id);
        setJustification(existingResponse.justification);
      } else {
        // Reset form
        setSelectedPath(null);
        setSelectedLevel(null);
        setJustification('');
      }
      setLevelViewStart(0);
    }
  }, [activeCategory, responses]);

  const loadData = async () => {
    setLoading(true);
    try {
      // Load assessment
      const assessmentData = await selfAssessmentService.getSelfAssessment(assessmentId);
      setAssessment(assessmentData);

      // Load catalog with details
      const catalogData = await adminService.getCatalog(assessmentData.catalog_id);
      setCatalog(catalogData);

      // Load responses
      const responsesData = await selfAssessmentService.getResponses(assessmentId);
      setResponses(responsesData);

      // Load completeness
      const completenessData = await selfAssessmentService.getCompleteness(assessmentId);
      setCompleteness(completenessData);
    } catch (error) {
      console.error('Error loading data:', error);
      notifications.show({
        title: 'Fehler',
        message: 'Daten konnten nicht geladen werden',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleSaveResponse = async () => {
    if (!selectedPath || !selectedLevel || !activeCategory) {
      notifications.show({
        title: 'Fehler',
        message: 'Bitte wählen Sie einen Pfad und ein Level',
        color: 'red',
      });
      return;
    }

    if (justification.length < 150) {
      notifications.show({
        title: 'Fehler',
        message: 'Die Begründung muss mindestens 150 Zeichen enthalten',
        color: 'red',
      });
      return;
    }

    setSaving(true);
    try {
      await selfAssessmentService.saveResponse(assessmentId, {
        category_id: parseInt(activeCategory),
        path_id: selectedPath,
        level_id: selectedLevel,
        justification,
      });

      // Reload responses and completeness
      const [responsesData, completenessData] = await Promise.all([
        selfAssessmentService.getResponses(assessmentId),
        selfAssessmentService.getCompleteness(assessmentId),
      ]);
      
      setResponses(responsesData);
      setCompleteness(completenessData);

      notifications.show({
        title: 'Gespeichert',
        message: 'Ihre Antwort wurde gespeichert',
        color: 'green',
      });
    } catch (error) {
      console.error('Error saving response:', error);
      notifications.show({
        title: 'Fehler',
        message: 'Antwort konnte nicht gespeichert werden',
        color: 'red',
      });
    } finally {
      setSaving(false);
    }
  };

  const handleSubmitAssessment = async () => {
    if (!completeness?.is_complete) {
      notifications.show({
        title: 'Fehler',
        message: 'Bitte füllen Sie alle Kategorien aus, bevor Sie einreichen',
        color: 'red',
      });
      return;
    }

    try {
      await selfAssessmentService.submitAssessment(assessmentId);
      notifications.show({
        title: 'Eingereicht',
        message: 'Ihre Selbsteinschätzung wurde zur Review eingereicht',
        color: 'green',
      });
      navigate('/self-assessments');
    } catch (error) {
      console.error('Error submitting assessment:', error);
      notifications.show({
        title: 'Fehler',
        message: 'Selbsteinschätzung konnte nicht eingereicht werden',
        color: 'red',
      });
    }
  };

  const getCurrentCategory = (): CategoryWithPaths | null => {
    if (!catalog?.categories || !activeCategory) return null;
    return catalog.categories.find(c => c.id.toString() === activeCategory) || null;
  };

  const getVisibleLevels = (): Level[] => {
    if (!catalog?.levels) return [];
    const sorted = [...catalog.levels].sort((a, b) => a.level_number - b.level_number);
    return sorted.slice(levelViewStart, levelViewStart + LEVELS_PER_VIEW);
  };

  const canScrollLeft = (): boolean => {
    return levelViewStart > 0;
  };

  const canScrollRight = (): boolean => {
    return catalog?.levels ? levelViewStart + LEVELS_PER_VIEW < catalog.levels.length : false;
  };

  const scrollLevels = (direction: 'left' | 'right') => {
    if (direction === 'left' && canScrollLeft()) {
      setLevelViewStart(prev => Math.max(0, prev - 1));
    } else if (direction === 'right' && canScrollRight()) {
      setLevelViewStart(prev => prev + 1);
    }
  };

  const getLevelDescription = (pathId: number, levelId: number): string => {
    const path = getCurrentCategory()?.paths?.find(p => p.id === pathId);
    const desc = path?.descriptions?.find(d => d.level_id === levelId);
    return desc?.description || 'Keine Beschreibung verfügbar';
  };

  const isCategoryComplete = (categoryId: number): boolean => {
    return responses.some(r => r.category_id === categoryId);
  };

  const getProgressColor = (): string => {
    if (!completeness) return 'blue';
    if (completeness.percent_complete === 100) return 'green';
    if (completeness.percent_complete >= 50) return 'yellow';
    return 'red';
  };

  if (loading) {
    return (
      <Center style={{ height: '80vh' }}>
        <Loader size="lg" />
      </Center>
    );
  }

  if (!assessment || !catalog) {
    return (
      <Container>
        <Alert icon={<IconAlertCircle />} title="Fehler" color="red">
          Selbsteinschätzung nicht gefunden
        </Alert>
      </Container>
    );
  }

  if (assessment.status !== 'draft') {
    return (
      <Container>
        <Alert icon={<IconAlertCircle />} title="Nicht editierbar" color="yellow">
          Diese Selbsteinschätzung kann nicht mehr bearbeitet werden (Status: {assessment.status})
        </Alert>
      </Container>
    );
  }

  const visibleLevels = getVisibleLevels();

  return (
    <Container size="xl" py="xl">
      <Button
        leftSection={<IconArrowLeft size={16} />}
        variant="subtle"
        onClick={() => navigate('/self-assessments')}
        mb="md"
      >
        Zurück zu Selbsteinschätzungen
      </Button>

      <Paper shadow="sm" p="lg" mb="xl">
        <Stack gap="md">
          <Group justify="space-between">
            <div>
              <Title order={2}>{catalog.name}</Title>
              <Text c="dimmed" size="sm">
                Selbsteinschätzung erstellt am {new Date(assessment.created_at).toLocaleDateString()}
              </Text>
            </div>
            <Badge size="lg" color={assessment.status === 'draft' ? 'blue' : 'gray'}>
              {assessment.status}
            </Badge>
          </Group>

          {completeness && (
            <div>
              <Group justify="space-between" mb="xs">
                <Text size="sm" fw={500}>
                  Fortschritt: {completeness.completed_categories} von {completeness.total_categories} Kategorien
                </Text>
                <Text size="sm" fw={500}>
                  {completeness.percent_complete.toFixed(0)}%
                </Text>
              </Group>
              <Progress value={completeness.percent_complete} color={getProgressColor()} size="lg" />
            </div>
          )}

          <Button
            size="lg"
            disabled={!completeness?.is_complete}
            onClick={handleSubmitAssessment}
            fullWidth
          >
            Selbsteinschätzung einreichen
          </Button>
        </Stack>
      </Paper>

      <Tabs value={activeCategory} onChange={setActiveCategory}>
        <Tabs.List>
          {catalog.categories?.map((category) => (
            <Tabs.Tab
              key={category.id}
              value={category.id.toString()}
              rightSection={
                isCategoryComplete(category.id) ? (
                  <IconCheck size={16} color="green" />
                ) : null
              }
            >
              {category.name}
            </Tabs.Tab>
          ))}
        </Tabs.List>

        {catalog.categories?.map((category) => (
          <Tabs.Panel key={category.id} value={category.id.toString()} pt="xl">
            <Stack gap="lg">
              {/* Category Description */}
              {category.description && (
                <Alert color="blue" variant="light">
                  {category.description}
                </Alert>
              )}

              {/* Path Selection */}
              <Paper p="md" withBorder>
                <Title order={4} mb="md">
                  Wählen Sie Ihren Pfad:
                </Title>
                <Radio.Group value={selectedPath?.toString() || ''} onChange={(val) => setSelectedPath(parseInt(val))}>
                  <Stack gap="xs">
                    {category.paths?.map((path) => (
                      <Radio
                        key={path.id}
                        value={path.id.toString()}
                        label={
                          <div>
                            <Text fw={500}>{path.name}</Text>
                            {path.description && (
                              <Text size="sm" c="dimmed">
                                {path.description}
                              </Text>
                            )}
                          </div>
                        }
                      />
                    ))}
                  </Stack>
                </Radio.Group>
              </Paper>

              {/* Level Selection */}
              {selectedPath && (
                <Paper p="md" withBorder>
                  <Title order={4} mb="md">
                    Level-Vergleich:
                  </Title>

                  <Group justify="center" mb="md">
                    <ActionIcon
                      size="lg"
                      variant="subtle"
                      onClick={() => scrollLevels('left')}
                      disabled={!canScrollLeft()}
                    >
                      <IconChevronLeft />
                    </ActionIcon>

                    <Group gap="md" style={{ flex: 1 }} justify="center">
                      {visibleLevels.map((level) => (
                        <Card
                          key={level.id}
                          shadow="sm"
                          p="md"
                          withBorder
                          style={{
                            flex: 1,
                            maxWidth: '300px',
                            border: selectedLevel === level.id ? '2px solid var(--mantine-color-blue-6)' : undefined,
                          }}
                        >
                          <Stack gap="sm">
                            <div>
                              <Badge mb="xs">Level {level.level_number}</Badge>
                              <Text fw={600} size="lg">
                                {level.name}
                              </Text>
                            </div>

                            {level.description && (
                              <Text size="sm" c="dimmed">
                                {level.description}
                              </Text>
                            )}

                            <Text size="sm" style={{ minHeight: '100px' }}>
                              {getLevelDescription(selectedPath, level.id)}
                            </Text>

                            <Button
                              variant={selectedLevel === level.id ? 'filled' : 'outline'}
                              onClick={() => setSelectedLevel(level.id)}
                              fullWidth
                            >
                              {selectedLevel === level.id ? 'Gewählt' : 'Wählen'}
                            </Button>
                          </Stack>
                        </Card>
                      ))}
                    </Group>

                    <ActionIcon
                      size="lg"
                      variant="subtle"
                      onClick={() => scrollLevels('right')}
                      disabled={!canScrollRight()}
                    >
                      <IconChevronRight />
                    </ActionIcon>
                  </Group>

                  <Text size="sm" c="dimmed" ta="center">
                    Level {levelViewStart + 1} bis {Math.min(levelViewStart + LEVELS_PER_VIEW, catalog.levels?.length || 0)}{' '}
                    von {catalog.levels?.length || 0}
                  </Text>
                </Paper>
              )}

              {/* Justification */}
              {selectedLevel && (
                <Paper p="md" withBorder>
                  <Title order={4} mb="md">
                    Begründung (mindestens 150 Zeichen):
                  </Title>

                  <Textarea
                    placeholder="Beschreiben Sie, warum Sie sich auf diesem Level einschätzen..."
                    minRows={6}
                    value={justification}
                    onChange={(e) => setJustification(e.target.value)}
                    error={justification.length > 0 && justification.length < 150 ? `Noch ${150 - justification.length} Zeichen erforderlich` : undefined}
                  />

                  <Group justify="space-between" mt="xs">
                    <Text size="sm" c={justification.length >= 150 ? 'green' : 'dimmed'}>
                      {justification.length} / 150 Zeichen
                    </Text>

                    <Button
                      onClick={handleSaveResponse}
                      disabled={justification.length < 150 || saving}
                      loading={saving}
                    >
                      Speichern
                    </Button>
                  </Group>
                </Paper>
              )}
            </Stack>
          </Tabs.Panel>
        ))}
      </Tabs>
    </Container>
  );
}
