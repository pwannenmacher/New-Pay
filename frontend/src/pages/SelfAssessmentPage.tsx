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
  Timeline,
  Divider,
  Modal,
} from '@mantine/core';
import {
  IconChevronLeft,
  IconChevronRight,
  IconCheck,
  IconAlertCircle,
  IconArrowLeft,
  IconClock,
  IconFileCheck,
  IconMessageCircle,
  IconArchive,
  IconX,
  IconSend,
  IconEdit,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { selfAssessmentService } from '../services/selfAssessment';
import catalogService from '../services/catalog';
import { WeightedScoreDisplay } from '../components/WeightedScoreDisplay';
import type {
  SelfAssessment,
  CatalogWithDetails,
  AssessmentResponseWithDetails,
  AssessmentCompleteness,
  CategoryWithPaths,
  Level,
} from '../types';

const statusConfig = {
  draft: { label: 'Entwurf', color: 'gray', icon: IconClock },
  submitted: { label: 'Eingereicht', color: 'blue', icon: IconFileCheck },
  in_review: { label: 'In Prüfung', color: 'yellow', icon: IconClock },
  reviewed: { label: 'Geprüft', color: 'orange', icon: IconCheck },
  discussion: { label: 'Besprechung', color: 'violet', icon: IconMessageCircle },
  archived: { label: 'Archiviert', color: 'green', icon: IconArchive },
  closed: { label: 'Geschlossen', color: 'red', icon: IconX },
};

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
  const [updating, setUpdating] = useState(false);
  const [submitModalOpen, setSubmitModalOpen] = useState(false);

  const [activeCategory, setActiveCategory] = useState<string | null>(null);
  const [selectedPath, setSelectedPath] = useState<number | null>(null);
  const [selectedLevel, setSelectedLevel] = useState<number | null>(null);
  const [justification, setJustification] = useState('');
  const [levelViewStart, setLevelViewStart] = useState(0);

  const LEVELS_PER_VIEW = 3;

  const isReadOnly = assessment?.status !== 'draft';

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
    if (activeCategory && responses && responses.length > 0) {
      const categoryId = parseInt(activeCategory);
      const existingResponse = responses.find((r) => r.category_id === categoryId);

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
      const catalogData = await catalogService.getCatalog(assessmentData.catalog_id);
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
      setSubmitModalOpen(false);
      return;
    }

    try {
      await selfAssessmentService.submitAssessment(assessmentId);
      notifications.show({
        title: 'Eingereicht',
        message: 'Ihre Selbsteinschätzung wurde zur Review eingereicht',
        color: 'green',
      });
      setSubmitModalOpen(false);
      await loadData(); // Reload to show updated status
    } catch (error) {
      console.error('Error submitting assessment:', error);
      notifications.show({
        title: 'Fehler',
        message: 'Selbsteinschätzung konnte nicht eingereicht werden',
        color: 'red',
      });
      setSubmitModalOpen(false);
    }
  };

  const handleStatusChange = async (newStatus: string) => {
    if (!assessment) return;

    try {
      setUpdating(true);
      await selfAssessmentService.updateStatus(assessment.id, newStatus);
      notifications.show({
        title: 'Erfolg',
        message:
          newStatus === 'submitted'
            ? 'Selbsteinschätzung wurde eingereicht'
            : newStatus === 'closed'
              ? 'Selbsteinschätzung wurde storniert'
              : 'Status wurde aktualisiert',
        color: 'green',
      });
      await loadData();
    } catch (error: any) {
      console.error('Error updating status:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Status konnte nicht aktualisiert werden',
        color: 'red',
      });
    } finally {
      setUpdating(false);
    }
  };

  const canReopen = () => {
    if (!assessment || assessment.status !== 'closed' || !assessment.closed_at) {
      return false;
    }
    const closedAt = new Date(assessment.closed_at);
    const now = new Date();
    const hoursSinceClosed = (now.getTime() - closedAt.getTime()) / (1000 * 60 * 60);
    return hoursSinceClosed < 24;
  };

  const getRemainingReopenTime = () => {
    if (!assessment?.closed_at) return null;
    const closedAt = new Date(assessment.closed_at);
    const deadline = new Date(closedAt.getTime() + 24 * 60 * 60 * 1000);
    const now = new Date();
    const remaining = deadline.getTime() - now.getTime();

    if (remaining <= 0) return null;

    const hours = Math.floor(remaining / (1000 * 60 * 60));
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
  };

  const handleReopen = async () => {
    if (!assessment?.previous_status) {
      notifications.show({
        title: 'Fehler',
        message: 'Vorheriger Status nicht gefunden',
        color: 'red',
      });
      return;
    }

    try {
      setUpdating(true);
      await selfAssessmentService.updateStatus(assessment.id, assessment.previous_status);
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde wiedereröffnet',
        color: 'green',
      });
      await loadData();
    } catch (error: any) {
      console.error('Error reopening assessment:', error);
      notifications.show({
        title: 'Fehler',
        message:
          error.response?.data?.error || 'Selbsteinschätzung konnte nicht wiedereröffnet werden',
        color: 'red',
      });
    } finally {
      setUpdating(false);
    }
  };

  const getCurrentCategory = (): CategoryWithPaths | null => {
    if (!catalog?.categories || !activeCategory) return null;
    return catalog.categories.find((c) => c.id.toString() === activeCategory) || null;
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
      setLevelViewStart((prev) => Math.max(0, prev - 1));
    } else if (direction === 'right' && canScrollRight()) {
      setLevelViewStart((prev) => prev + 1);
    }
  };

  const getLevelDescription = (pathId: number, levelId: number): string => {
    const path = getCurrentCategory()?.paths?.find((p) => p.id === pathId);
    const desc = path?.descriptions?.find((d) => d.level_id === levelId);
    return desc?.description || 'Keine Beschreibung verfügbar';
  };

  const isCategoryComplete = (categoryId: number): boolean => {
    return responses.some((r) => r.category_id === categoryId);
  };

  const getProgressColor = (): string => {
    if (!completeness) return 'blue';
    if (completeness.percent_complete === 100) return 'green';
    if (completeness.percent_complete >= 50) return 'yellow';
    return 'red';
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return null;
    return new Date(dateString).toLocaleString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusBadge = (status: string) => {
    const config = statusConfig[status as keyof typeof statusConfig] || {
      label: status,
      color: 'gray',
      icon: IconClock,
    };
    const Icon = config.icon;
    return (
      <Badge size="lg" color={config.color} leftSection={<Icon size={14} />}>
        {config.label}
      </Badge>
    );
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

  const visibleLevels = getVisibleLevels();
  const canSubmit = assessment.status === 'draft';

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
                Erstellt am {new Date(assessment.created_at).toLocaleDateString('de-DE')}
                {assessment.user_email && ` • ${assessment.user_email}`}
              </Text>
            </div>
            <Stack align="flex-end" gap="xs">
              {getStatusBadge(assessment.status)}
              <WeightedScoreDisplay assessmentId={assessment.id} />
            </Stack>
          </Group>

          {/* Status-dependent alerts and actions */}
          {canSubmit && (
            <>
              {completeness && (
                <div>
                  <Group justify="space-between" mb="xs">
                    <Text size="sm" fw={500}>
                      Fortschritt: {completeness.completed_categories} von{' '}
                      {completeness.total_categories} Kategorien
                    </Text>
                    <Text size="sm" fw={500}>
                      {completeness.percent_complete.toFixed(0)}%
                    </Text>
                  </Group>
                  <Progress
                    value={completeness.percent_complete}
                    color={getProgressColor()}
                    size="lg"
                  />
                </div>
              )}

              <Divider />

              <Alert icon={<IconAlertCircle size={16} />} color="blue">
                Diese Selbsteinschätzung befindet sich noch im Entwurf. Sie können sie bearbeiten,
                zur Prüfung einreichen oder stornieren.
              </Alert>

              <Group>
                <Button
                  size="lg"
                  leftSection={<IconSend size={16} />}
                  disabled={!completeness?.is_complete}
                  onClick={() => setSubmitModalOpen(true)}
                  color="blue"
                >
                  Zur Prüfung einreichen
                </Button>
                <Button
                  size="lg"
                  leftSection={<IconX size={16} />}
                  onClick={() => handleStatusChange('closed')}
                  loading={updating}
                  variant="light"
                  color="red"
                >
                  Stornieren
                </Button>
              </Group>

              {!completeness?.is_complete && (
                <Alert icon={<IconAlertCircle size={16} />} color="orange">
                  Sie müssen alle Kategorien ausfüllen, bevor Sie die Selbsteinschätzung einreichen
                  können.
                </Alert>
              )}
            </>
          )}

          {!canSubmit && (
            <>
              <Alert icon={<IconAlertCircle size={16} />} color="yellow">
                Diese Selbsteinschätzung ist nicht mehr editierbar (Status:{' '}
                {statusConfig[assessment.status as keyof typeof statusConfig]?.label ||
                  assessment.status}
                ). Sie können die ausgefüllten Antworten ansehen, aber keine Änderungen mehr
                vornehmen.
              </Alert>
            </>
          )}

          {canReopen() && (
            <>
              <Alert icon={<IconAlertCircle size={16} />} color="orange">
                Diese Selbsteinschätzung wurde geschlossen. Sie können sie innerhalb von 24 Stunden
                nach dem Schließen wiedereröffnen.
                {getRemainingReopenTime() && (
                  <Text size="sm" mt="xs" fw={500}>
                    Verbleibende Zeit: {getRemainingReopenTime()}
                  </Text>
                )}
              </Alert>
              <Group>
                <Button
                  leftSection={<IconEdit size={16} />}
                  onClick={handleReopen}
                  loading={updating}
                  color="orange"
                  variant="filled"
                >
                  Wiedereröffnen
                </Button>
              </Group>
            </>
          )}
        </Stack>
      </Paper>

      {/* Status history for non-draft assessments */}
      {!canSubmit && (
        <Paper shadow="sm" p="md" withBorder mb="xl">
          <Title order={3} mb="md">
            Status-Historie
          </Title>
          <Timeline active={-1} bulletSize={24} lineWidth={2}>
            <Timeline.Item bullet={<IconClock size={12} />} title="Erstellt">
              <Text c="dimmed" size="sm">
                {formatDate(assessment.created_at)}
              </Text>
            </Timeline.Item>

            {assessment.submitted_at && (
              <Timeline.Item bullet={<IconFileCheck size={12} />} title="Eingereicht">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.submitted_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.in_review_at && (
              <Timeline.Item bullet={<IconClock size={12} />} title="In Prüfung">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.in_review_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.reviewed_at && (
              <Timeline.Item bullet={<IconCheck size={12} />} title="Geprüft">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.reviewed_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.discussion_started_at && (
              <Timeline.Item bullet={<IconMessageCircle size={12} />} title="Besprechung">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.discussion_started_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.archived_at && (
              <Timeline.Item bullet={<IconArchive size={12} />} title="Archiviert">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.archived_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.closed_at && (
              <Timeline.Item bullet={<IconX size={12} />} title="Geschlossen">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.closed_at)}
                </Text>
              </Timeline.Item>
            )}
          </Timeline>
        </Paper>
      )}

      <Tabs value={activeCategory} onChange={setActiveCategory}>
        <Tabs.List>
          {catalog.categories?.map((category) => (
            <Tabs.Tab
              key={category.id}
              value={category.id.toString()}
              rightSection={
                isCategoryComplete(category.id) ? <IconCheck size={16} color="green" /> : null
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
                  {isReadOnly ? 'Gewählter Pfad:' : 'Wählen Sie Ihren Pfad:'}
                </Title>
                <Radio.Group
                  value={selectedPath?.toString() || ''}
                  onChange={(val) => !isReadOnly && setSelectedPath(parseInt(val))}
                >
                  <Stack gap="xs">
                    {category.paths?.map((path) => (
                      <Radio
                        key={path.id}
                        value={path.id.toString()}
                        disabled={isReadOnly}
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
                    {isReadOnly ? 'Gewähltes Level:' : 'Level-Vergleich:'}
                  </Title>

                  {!isReadOnly && (
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
                              border:
                                selectedLevel === level.id
                                  ? '2px solid var(--mantine-color-blue-6)'
                                  : undefined,
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
                  )}

                  {!isReadOnly && (
                    <Text size="sm" c="dimmed" ta="center">
                      Level {levelViewStart + 1} bis{' '}
                      {Math.min(levelViewStart + LEVELS_PER_VIEW, catalog.levels?.length || 0)} von{' '}
                      {catalog.levels?.length || 0}
                    </Text>
                  )}

                  {isReadOnly && selectedLevel && (
                    <Card shadow="sm" p="md" withBorder>
                      <Stack gap="sm">
                        {(() => {
                          const level = catalog.levels?.find((l) => l.id === selectedLevel);
                          return level ? (
                            <>
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
                              <Text size="sm">{getLevelDescription(selectedPath, level.id)}</Text>
                            </>
                          ) : null;
                        })()}
                      </Stack>
                    </Card>
                  )}
                </Paper>
              )}

              {/* Justification */}
              {selectedLevel && (
                <Paper p="md" withBorder>
                  <Title order={4} mb="md">
                    Begründung{!isReadOnly && ' (mindestens 150 Zeichen)'}:
                  </Title>

                  <Textarea
                    placeholder="Beschreiben Sie, warum Sie sich auf diesem Level einschätzen..."
                    minRows={6}
                    value={justification}
                    onChange={(e) => setJustification(e.target.value)}
                    error={
                      !isReadOnly && justification.length > 0 && justification.length < 150
                        ? `Noch ${150 - justification.length} Zeichen erforderlich`
                        : undefined
                    }
                    readOnly={isReadOnly}
                    disabled={isReadOnly}
                  />

                  {!isReadOnly && (
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
                  )}
                </Paper>
              )}
            </Stack>
          </Tabs.Panel>
        ))}
      </Tabs>

      {/* Submit confirmation modal */}
      <Modal
        opened={submitModalOpen}
        onClose={() => setSubmitModalOpen(false)}
        title="Selbsteinschätzung einreichen"
      >
        <Stack gap="md">
          <Text>Möchten Sie Ihre Selbsteinschätzung wirklich zur Prüfung einreichen?</Text>
          <Alert icon={<IconAlertCircle size={16} />} color="blue">
            Nach dem Einreichen können Sie keine Änderungen mehr vornehmen.
          </Alert>
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setSubmitModalOpen(false)}>
              Abbrechen
            </Button>
            <Button onClick={handleSubmitAssessment} color="blue">
              Einreichen
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}
