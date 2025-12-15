import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Paper,
  Title,
  Text,
  Button,
  Grid,
  Radio,
  Group,
  Textarea,
  Badge,
  Stack,
  Alert,
  Loader,
  Center,
  Tabs,
} from '@mantine/core';
import {
  IconArrowLeft,
  IconCheck,
  IconAlertCircle,
  IconClock,
  IconFileCheck,
  IconMessageCircle,
  IconArchive,
  IconX,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { selfAssessmentService } from '../../services/selfAssessment';
import adminService from '../../services/admin';
import { useAuth } from '../../contexts/AuthContext';
import type {
  SelfAssessment,
  CatalogWithDetails,
  AssessmentResponseWithDetails,
} from '../../types';

const statusConfig = {
  submitted: { label: 'Eingereicht', color: 'blue', icon: IconFileCheck },
  in_review: { label: 'In Prüfung', color: 'yellow', icon: IconClock },
  reviewed: { label: 'Geprüft', color: 'orange', icon: IconCheck },
  discussion: { label: 'Besprechung', color: 'violet', icon: IconMessageCircle },
  archived: { label: 'Archiviert', color: 'green', icon: IconArchive },
  closed: { label: 'Geschlossen', color: 'red', icon: IconX },
};

export function ReviewAssessmentPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const assessmentId = parseInt(id || '0');

  const [assessment, setAssessment] = useState<SelfAssessment | null>(null);
  const [catalog, setCatalog] = useState<CatalogWithDetails | null>(null);
  const [userResponses, setUserResponses] = useState<AssessmentResponseWithDetails[]>([]);
  const [reviewerResponses, setReviewerResponses] = useState<Map<number, { path_id: number; level_id: number; justification: string }>>(new Map());
  const [selectedPaths, setSelectedPaths] = useState<Map<number, number>>(new Map());
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [activeCategory, setActiveCategory] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, [assessmentId]);

  useEffect(() => {
    if (catalog?.categories && catalog.categories.length > 0 && !activeCategory) {
      setActiveCategory(catalog.categories[0].id.toString());
    }
  }, [catalog]);

  const loadData = async () => {
    try {
      setLoading(true);
      const [assessmentData, responsesData] = await Promise.all([
        selfAssessmentService.getSelfAssessment(assessmentId),
        selfAssessmentService.getResponses(assessmentId),
      ]);

      setAssessment(assessmentData);
      setUserResponses(responsesData);

      if (assessmentData.catalog_id) {
        const catalogData = await adminService.getCatalog(assessmentData.catalog_id);
        setCatalog(catalogData);
      }

      // TODO: Load existing reviewer responses from backend
      // For now, initialize with user's choices
      const initialReviewerResponses = new Map();
      const initialSelectedPaths = new Map();
      responsesData.forEach((response) => {
        initialReviewerResponses.set(response.category_id, {
          path_id: response.path_id,
          level_id: response.level_id,
          justification: '',
        });
        initialSelectedPaths.set(response.category_id, response.path_id);
      });
      setReviewerResponses(initialReviewerResponses);
      setSelectedPaths(initialSelectedPaths);
    } catch (error: any) {
      console.error('Error loading data:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Daten konnten nicht geladen werden',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const config = statusConfig[status as keyof typeof statusConfig];
    if (!config) return <Badge>{status}</Badge>;

    const Icon = config.icon;
    return (
      <Badge color={config.color} leftSection={<Icon size={14} />}>
        {config.label}
      </Badge>
    );
  };

  const handleReviewerPathChange = (categoryId: number, pathId: number) => {
    setSelectedPaths(new Map(selectedPaths.set(categoryId, pathId)));
    const current = reviewerResponses.get(categoryId) || { path_id: pathId, level_id: 0, justification: '' };
    setReviewerResponses(new Map(reviewerResponses.set(categoryId, { ...current, path_id: pathId, level_id: 0 })));
  };

  const handleReviewerLevelChange = (categoryId: number, levelId: number) => {
    const pathId = selectedPaths.get(categoryId) || 0;
    const current = reviewerResponses.get(categoryId) || { path_id: pathId, level_id: levelId, justification: '' };
    setReviewerResponses(new Map(reviewerResponses.set(categoryId, { ...current, level_id: levelId })));
  };

  const handleReviewerJustificationChange = (categoryId: number, justification: string) => {
    const pathId = selectedPaths.get(categoryId) || 0;
    const current = reviewerResponses.get(categoryId) || { path_id: pathId, level_id: 0, justification };
    setReviewerResponses(new Map(reviewerResponses.set(categoryId, { ...current, justification })));
  };

  const getUserResponseForCategory = (categoryId: number) => {
    return userResponses.find((r) => r.category_id === categoryId);
  };

  const getReviewerResponseForCategory = (categoryId: number) => {
    return reviewerResponses.get(categoryId);
  };

  const isJustificationRequired = (categoryId: number): boolean => {
    const userResponse = getUserResponseForCategory(categoryId);
    const reviewerResponse = getReviewerResponseForCategory(categoryId);
    
    if (!userResponse || !reviewerResponse) return false;
    
    // Begründung erforderlich wenn Level ODER Pfad abweicht
    return userResponse.level_id !== reviewerResponse.level_id || userResponse.path_id !== reviewerResponse.path_id;
  };

  const isJustificationValid = (categoryId: number): boolean => {
    const reviewerResponse = getReviewerResponseForCategory(categoryId);
    
    if (!isJustificationRequired(categoryId)) return true;
    
    return (reviewerResponse?.justification?.length || 0) >= 50;
  };

  const canSaveReview = (): boolean => {
    if (!catalog?.categories) return false;
    
    return catalog.categories.every((category) => {
      const reviewerResponse = getReviewerResponseForCategory(category.id);
      if (!reviewerResponse || !reviewerResponse.level_id) return false;
      return isJustificationValid(category.id);
    });
  };

  const handleSaveReview = async () => {
    if (!canSaveReview()) {
      notifications.show({
        title: 'Validierung fehlgeschlagen',
        message: 'Bitte füllen Sie alle erforderlichen Felder aus.',
        color: 'red',
      });
      return;
    }

    try {
      setSaving(true);
      
      // TODO: Save reviewer responses to backend
      console.log('Saving reviewer responses:', Object.fromEntries(reviewerResponses));
      
      notifications.show({
        title: 'Erfolg',
        message: 'Review wurde gespeichert',
        color: 'green',
      });
      
      navigate('/review/open-assessments');
    } catch (error: any) {
      console.error('Error saving review:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Review konnte nicht gespeichert werden',
        color: 'red',
      });
    } finally {
      setSaving(false);
    }
  };

  const getLevelNameById = (levelId: number): string => {
    const level = catalog?.levels?.find((l) => l.id === levelId);
    return level ? `Level ${level.level_number}: ${level.name}` : 'Unbekannt';
  };

  const getPathNameById = (pathId: number): string => {
    const category = catalog?.categories?.find((c) =>
      c.paths?.some((p) => p.id === pathId)
    );
    const path = category?.paths?.find((p) => p.id === pathId);
    return path?.name || 'Unbekannt';
  };

  if (loading) {
    return (
      <Center style={{ height: '100vh' }}>
        <Loader size="xl" />
      </Center>
    );
  }

  if (!assessment || !catalog) {
    return (
      <Container size="xl" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title="Fehler" color="red">
          Selbsteinschätzung konnte nicht geladen werden.
        </Alert>
      </Container>
    );
  }

  // Verhindere dass User ihre eigenen Assessments prüfen
  if (user?.id === assessment.user_id) {
    return (
      <Container size="xl" py="xl">
        <Stack gap="lg">
          <Alert icon={<IconAlertCircle size={16} />} title="Zugriff verweigert" color="red">
            Sie können Ihre eigenen Selbsteinschätzungen nicht als Reviewer prüfen.
          </Alert>
          <Button
            variant="light"
            leftSection={<IconArrowLeft size={16} />}
            onClick={() => navigate('/review/open-assessments')}
          >
            Zurück zur Übersicht
          </Button>
        </Stack>
      </Container>
    );
  }

  const activeCategories = catalog.categories || [];
  const activeCategoryData = activeCategories.find((c) => c.id.toString() === activeCategory);

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <Button
            variant="subtle"
            leftSection={<IconArrowLeft size={16} />}
            onClick={() => navigate('/review/open-assessments')}
          >
            Zurück zur Übersicht
          </Button>
        </Group>

        <Paper shadow="sm" p="md" withBorder>
          <Stack gap="sm">
            <Group justify="space-between">
              <div>
                <Title order={2}>{catalog.name}</Title>
                <Text c="dimmed" size="sm">
                  Benutzer: {assessment.user_name || assessment.user_email || 'Unbekannt'}
                </Text>
              </div>
              {getStatusBadge(assessment.status)}
            </Group>
            {catalog.description && (
              <Text size="sm" c="dimmed">
                {catalog.description}
              </Text>
            )}
          </Stack>
        </Paper>

        {activeCategories.length > 0 && (
          <Tabs value={activeCategory} onChange={setActiveCategory}>
            <Tabs.List>
              {activeCategories.map((category) => {
                const reviewerResponse = getReviewerResponseForCategory(category.id);
                const hasResponse = !!reviewerResponse?.level_id;
                const isValid = isJustificationValid(category.id);

                return (
                  <Tabs.Tab
                    key={category.id}
                    value={category.id.toString()}
                    rightSection={
                      hasResponse ? (
                        isValid ? (
                          <IconCheck size={14} color="green" />
                        ) : (
                          <IconAlertCircle size={14} color="red" />
                        )
                      ) : null
                    }
                  >
                    {category.name}
                  </Tabs.Tab>
                );
              })}
            </Tabs.List>

            {activeCategories.map((category) => (
              <Tabs.Panel key={category.id} value={category.id.toString()} pt="md">
                {activeCategoryData?.id === category.id && (
                  <Grid>
                    {/* Left Column: User Assessment */}
                    <Grid.Col span={6}>
                      <Paper p="md" withBorder style={{ height: '100%' }}>
                        <Stack gap="md">
                          <Title order={4}>Selbsteinschätzung des Benutzers</Title>
                          
                          {(() => {
                            const userResponse = getUserResponseForCategory(category.id);
                            
                            if (!userResponse) {
                              return (
                                <Alert icon={<IconAlertCircle size={16} />} color="yellow">
                                  Keine Antwort vorhanden
                                </Alert>
                              );
                            }

                            return (
                              <Stack gap="md">
                                <div>
                                  <Text size="sm" fw={600} c="dimmed" mb="xs">
                                    Entwicklungspfad
                                  </Text>
                                  <Badge size="lg" variant="light">
                                    {getPathNameById(userResponse.path_id)}
                                  </Badge>
                                </div>

                                <div>
                                  <Text size="sm" fw={600} c="dimmed" mb="xs">
                                    Gewähltes Level
                                  </Text>
                                  <Badge size="lg" color="blue">
                                    {getLevelNameById(userResponse.level_id)}
                                  </Badge>
                                </div>

                                <div>
                                  <Text size="sm" fw={600} c="dimmed" mb="xs">
                                    Begründung
                                  </Text>
                                  <Paper p="sm" withBorder bg="gray.0">
                                    <Text size="sm">
                                      {userResponse.justification || 'Keine Begründung angegeben'}
                                    </Text>
                                  </Paper>
                                </div>
                              </Stack>
                            );
                          })()}
                        </Stack>
                      </Paper>
                    </Grid.Col>

                    {/* Right Column: Reviewer Assessment */}
                    <Grid.Col span={6}>
                      <Paper p="md" withBorder style={{ height: '100%' }}>
                        <Stack gap="md">
                          <Title order={4}>Ihre Bewertung als Reviewer</Title>
                          
                          {(() => {
                            const userResponse = getUserResponseForCategory(category.id);
                            const reviewerResponse = getReviewerResponseForCategory(category.id);
                            
                            if (!userResponse) {
                              return (
                                <Alert icon={<IconAlertCircle size={16} />} color="yellow">
                                  Keine Benutzereingabe vorhanden
                                </Alert>
                              );
                            }

                            const requiresJustification = isJustificationRequired(category.id);
                            const justificationValid = isJustificationValid(category.id);

                            const selectedPathId = selectedPaths.get(category.id) || userResponse.path_id;
                            const availablePaths = category.paths || [];

                            return (
                              <Stack gap="md">
                                <div>
                                  <Text size="sm" fw={600} mb="xs">
                                    Entwicklungspfad
                                  </Text>
                                  {availablePaths.length > 1 ? (
                                    <Radio.Group
                                      value={selectedPathId.toString()}
                                      onChange={(value) => handleReviewerPathChange(category.id, parseInt(value))}
                                    >
                                      <Stack gap="xs">
                                        {availablePaths.map((path) => (
                                          <Radio
                                            key={path.id}
                                            value={path.id.toString()}
                                            label={
                                              <Group gap="xs">
                                                <Text fw={600}>{path.name}</Text>
                                                {path.id === userResponse.path_id && (
                                                  <Badge size="xs" color="blue">Benutzer-Wahl</Badge>
                                                )}
                                              </Group>
                                            }
                                          />
                                        ))}
                                      </Stack>
                                    </Radio.Group>
                                  ) : (
                                    <Badge size="lg" variant="light">
                                      {getPathNameById(selectedPathId)}
                                    </Badge>
                                  )}
                                </div>

                                <div>
                                  <Text size="sm" fw={600} mb="xs">
                                    Level-Bewertung
                                  </Text>
                                  <Radio.Group
                                    value={reviewerResponse?.level_id?.toString() || ''}
                                    onChange={(value) => handleReviewerLevelChange(category.id, parseInt(value))}
                                  >
                                    <Stack gap="xs">
                                      {catalog.levels?.map((level) => (
                                        <Radio
                                          key={level.id}
                                          value={level.id.toString()}
                                          label={
                                            <div>
                                              <Text fw={600}>
                                                Level {level.level_number}: {level.name}
                                              </Text>
                                              {level.description && (
                                                <Text size="xs" c="dimmed">
                                                  {level.description}
                                                </Text>
                                              )}
                                            </div>
                                          }
                                        />
                                      ))}
                                    </Stack>
                                  </Radio.Group>
                                </div>

                                <div>
                                  <Group justify="space-between" mb="xs">
                                    <Text size="sm" fw={600}>
                                      Begründung
                                      {requiresJustification && (
                                        <Text component="span" c="red" ml={4}>
                                          *
                                        </Text>
                                      )}
                                    </Text>
                                    {requiresJustification && (
                                      <Text size="xs" c={justificationValid ? 'green' : 'red'}>
                                        {reviewerResponse?.justification?.length || 0} / 50 Zeichen
                                      </Text>
                                    )}
                                  </Group>
                                  <Textarea
                                    placeholder={
                                      requiresJustification
                                        ? 'Begründung erforderlich (mindestens 50 Zeichen), da Sie vom Benutzer-Level abweichen'
                                        : 'Optional: Begründung für Ihre Bewertung'
                                    }
                                    value={reviewerResponse?.justification || ''}
                                    onChange={(e) =>
                                      handleReviewerJustificationChange(category.id, e.target.value)
                                    }
                                    minRows={4}
                                    error={
                                      requiresJustification && !justificationValid
                                        ? 'Mindestens 50 Zeichen erforderlich'
                                        : undefined
                                    }
                                  />
                                  {requiresJustification && (
                                    <Text size="xs" c="dimmed" mt="xs">
                                      Sie haben ein anderes Level als der Benutzer gewählt. Bitte begründen Sie
                                      Ihre Abweichung ausführlich.
                                    </Text>
                                  )}
                                </div>
                              </Stack>
                            );
                          })()}
                        </Stack>
                      </Paper>
                    </Grid.Col>
                  </Grid>
                )}
              </Tabs.Panel>
            ))}
          </Tabs>
        )}

        <Paper p="md" withBorder>
          <Group justify="space-between">
            <Button
              variant="subtle"
              leftSection={<IconArrowLeft size={16} />}
              onClick={() => navigate('/review/open-assessments')}
            >
              Zurück zur Übersicht
            </Button>
            <Button
              leftSection={<IconCheck size={16} />}
              onClick={handleSaveReview}
              loading={saving}
              disabled={!canSaveReview()}
            >
              Review speichern
            </Button>
          </Group>
        </Paper>
      </Stack>
    </Container>
  );
}
