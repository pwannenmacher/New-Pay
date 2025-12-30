import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Paper,
  Title,
  Text,
  Button,
  Grid,
  Stack,
  Alert,
  Loader,
  Center,
  Tabs,
  Badge,
  Group,
  Textarea,
  Select,
  Tooltip,
  Radio,
  useMantineColorScheme,
} from '@mantine/core';
import { IconArrowLeft, IconAlertCircle, IconCheck } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import consolidationService, { type ConsolidationData, type ConsolidationOverride } from '../../services/consolidation';
import { useAuth } from '../../contexts/AuthContext';

export function ReviewConsolidationPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const { colorScheme } = useMantineColorScheme();
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<ConsolidationData | null>(null);
  const [activeCategory, setActiveCategory] = useState<string>('0');
  const [overrideData, setOverrideData] = useState<{ [key: number]: Partial<ConsolidationOverride> }>({});
  const [categoryComments, setCategoryComments] = useState<{ [key: number]: string }>({});
  const [finalComment, setFinalComment] = useState<string>('');

  // Check if review is in read-only mode (status is 'reviewed' or 'discussion')
  const isReadOnly = data?.assessment?.status === 'reviewed' || data?.assessment?.status === 'discussion';
  
  // Check if within 1 hour of reviewed_at for revocation
  const canRevokeApprovals = data?.assessment?.status === 'reviewed' && data?.assessment?.reviewed_at
    ? new Date().getTime() - new Date(data.assessment.reviewed_at).getTime() < 60 * 60 * 1000
    : false;

  useEffect(() => {
    loadData();
  }, [id]);

  const loadData = async () => {
    try {
      setLoading(true);
      const consolidationData = await consolidationService.getConsolidationData(parseInt(id!));
      
      if (!consolidationData) {
        throw new Error('Keine Daten vom Server erhalten');
      }
      
      setData(consolidationData);
      
      // Initialize final comment if exists
      if (consolidationData.final_consolidation?.comment) {
        setFinalComment(consolidationData.final_consolidation.comment);
      }
      
      // Initialize override data with existing overrides
      const overrides: { [key: number]: Partial<ConsolidationOverride> } = {};
      if (consolidationData.overrides) {
        consolidationData.overrides.forEach(override => {
          overrides[override.category_id] = override;
        });
      }
      setOverrideData(overrides);

      // Initialize category comments
      const comments: { [key: number]: string } = {};
      if (consolidationData.category_discussion_comments) {
        consolidationData.category_discussion_comments.forEach(comment => {
          comments[comment.category_id] = comment.comment;
        });
      }
      setCategoryComments(comments);
    } catch (error: any) {
      console.error('Error loading consolidation data:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || error.message || 'Fehler beim Laden der Konsolidierungsdaten',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleSaveOverride = async (categoryId: number) => {
    const override = overrideData[categoryId];
    if (!override || !override.path_id || !override.level_id || !override.justification) {
      notifications.show({
        title: 'Fehler',
        message: 'Bitte wählen Sie Pfad, Stufe und geben Sie eine Begründung ein',
        color: 'red',
      });
      return;
    }

    try {
      await consolidationService.createOrUpdateOverride(parseInt(id!), {
        assessment_id: parseInt(id!),
        category_id: categoryId,
        path_id: override.path_id,
        level_id: override.level_id,
        justification: override.justification,
      });

      notifications.show({
        title: 'Erfolg',
        message: 'Überschreibung gespeichert',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Speichern der Überschreibung',
        color: 'red',
      });
    }
  };

  const handleApproveOverride = async (categoryId: number) => {
    try {
      await consolidationService.approveOverride(parseInt(id!), categoryId);

      notifications.show({
        title: 'Erfolg',
        message: 'Überschreibung bestätigt',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Bestätigen der Überschreibung',
        color: 'red',
      });
    }
  };

  const handleApproveAveraged = async (categoryId: number) => {
    try {
      await consolidationService.approveAveragedResponse(parseInt(id!), categoryId);

      notifications.show({
        title: 'Erfolg',
        message: 'Gemittelte Bewertung bestätigt',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Bestätigen der gemittelten Bewertung',
        color: 'red',
      });
    }
  };

  const handleDeleteOverride = async (categoryId: number) => {
    try {
      await consolidationService.deleteOverride(parseInt(id!), categoryId);

      notifications.show({
        title: 'Erfolg',
        message: 'Anpassung gelöscht',
        color: 'green',
      });

      // Reset override data for this category
      setOverrideData(prev => {
        const newData = { ...prev };
        delete newData[categoryId];
        return newData;
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Löschen der Anpassung',
        color: 'red',
      });
    }
  };

  const handleSaveFinalComment = async () => {
    if (!finalComment.trim()) {
      notifications.show({
        title: 'Fehler',
        message: 'Bitte geben Sie einen Abschluss-Kommentar ein',
        color: 'red',
      });
      return;
    }

    try {
      await consolidationService.saveFinalConsolidation(parseInt(id!), finalComment);

      notifications.show({
        title: 'Erfolg',
        message: 'Abschluss-Kommentar gespeichert',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Speichern des Abschluss-Kommentars',
        color: 'red',
      });
    }
  };

  const handleSaveCategoryComment = async (categoryId: number) => {
    const comment = categoryComments[categoryId];
    if (!comment || !comment.trim()) {
      notifications.show({
        title: 'Fehler',
        message: 'Bitte geben Sie einen Kommentar ein',
        color: 'red',
      });
      return;
    }

    try {
      await consolidationService.saveCategoryDiscussionComment(parseInt(id!), categoryId, comment);

      notifications.show({
        title: 'Erfolg',
        message: 'Kommentar gespeichert',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Speichern des Kommentars',
        color: 'red',
      });
    }
  };

  const handleRevokeOverrideApproval = async (categoryId: number) => {
    try {
      await consolidationService.revokeOverrideApproval(parseInt(id!), categoryId);

      notifications.show({
        title: 'Erfolg',
        message: 'Bestätigung zurückgenommen',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Zurücknehmen der Bestätigung',
        color: 'red',
      });
    }
  };

  const handleRevokeAveragedApproval = async (categoryId: number) => {
    try {
      await consolidationService.revokeAveragedApproval(parseInt(id!), categoryId);

      notifications.show({
        title: 'Erfolg',
        message: 'Bestätigung zurückgenommen',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Zurücknehmen der Bestätigung',
        color: 'red',
      });
    }
  };

  const handleRevokeFinalApproval = async () => {
    try {
      await consolidationService.revokeFinalApproval(parseInt(id!));

      notifications.show({
        title: 'Erfolg',
        message: 'Bestätigung zurückgenommen',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Zurücknehmen der Bestätigung',
        color: 'red',
      });
    }
  };

  const handleApproveFinal = async () => {
    try {
      await consolidationService.approveFinalConsolidation(parseInt(id!));

      notifications.show({
        title: 'Erfolg',
        message: 'Abschluss bestätigt',
        color: 'green',
      });

      loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Bestätigen des Abschlusses',
        color: 'red',
      });
    }
  };

  const hasOverrideChanged = (categoryId: number, existingOverride: ConsolidationOverride | undefined): boolean => {
    if (!existingOverride) return true;
    const current = overrideData[categoryId];
    if (!current) return false;
    
    return current.path_id !== existingOverride.path_id ||
           current.level_id !== existingOverride.level_id ||
           current.justification !== existingOverride.justification;
  };

  const isCurrentUserAuthor = (override: ConsolidationOverride | undefined): boolean => {
    if (!override || !user?.id) return false;
    return override.created_by_user_id === user.id;
  };

  const isCategoryApproved = (categoryId: number): boolean => {
    // Check if override exists and is approved (1+ approval)
    const override = data?.overrides?.find(o => o.category_id === categoryId);
    if (override && override.is_approved) {
      return true;
    }

    // Check if averaged response has 2+ approvals
    const averaged = data?.averaged_responses?.find(a => a.category_id === categoryId);
    if (averaged && averaged.is_approved) {
      return true;
    }

    return false;
  };

  if (loading) {
    return (
      <Center h={400}>
        <Loader size="lg" />
      </Center>
    );
  }

  if (!data) {
    return (
      <Container size="lg" py="xl">
        <Alert icon={<IconAlertCircle size={16} />} title="Fehler" color="red">
          Konsolidierungsdaten konnten nicht geladen werden
        </Alert>
      </Container>
    );
  }

  const categories = data.catalog.categories || [];
  const sortedCategories = [...categories].sort((a, b) => a.sort_order - b.sort_order);

  // Calculate weighted overall average
  const calculateOverallAverage = () => {
    let totalWeightedScore = 0;
    let totalWeight = 0;

    sortedCategories.forEach(category => {
      const averaged = data.averaged_responses.find(r => r.category_id === category.id);
      const override = data.overrides.find(o => o.category_id === category.id);
      
      // Use override if it exists (TODO: check if approved when approval system is implemented)
      if (override) {
        // Find the level number for the override
        const overrideLevel = data.catalog.levels.find((l: any) => l.id === override.level_id);
        if (overrideLevel) {
          totalWeightedScore += overrideLevel.level_number * category.weight;
          totalWeight += category.weight;
        }
      } else if (averaged) {
        // Use averaged reviewer response if no override exists
        totalWeightedScore += averaged.average_level_number * category.weight;
        totalWeight += category.weight;
      }
    });

    if (totalWeight === 0) return { number: 0, name: '-' };
    
    const overallAverage = totalWeightedScore / totalWeight;
    
    // Find closest level name
    const closestLevel = data.catalog.levels
      .map((l: any) => ({ ...l, diff: Math.abs(l.level_number - overallAverage) }))
      .sort((a: any, b: any) => a.diff - b.diff)[0];
    
    return {
      number: Math.round(overallAverage * 100) / 100,
      name: closestLevel?.name || '-'
    };
  };

  const overallAverage = calculateOverallAverage();

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <Button
            leftSection={<IconArrowLeft size={16} />}
            variant="subtle"
            onClick={() => navigate('/review/open-assessments')}
          >
            Zurück zur Übersicht
          </Button>
        </Group>

        {isReadOnly && (
          <Alert color="blue" title="Konsolidierung abgeschlossen">
            Die Review-Konsolidierung wurde abgeschlossen. Alle Daten sind schreibgeschützt.
            {canRevokeApprovals && (
              <Text size="sm" mt="xs">
                Sie können Ihre Bestätigungen noch innerhalb der nächsten Stunde zurücknehmen.
              </Text>
            )}
          </Alert>
        )}

        <Title order={2}>Review-Konsolidierung</Title>

        <Alert color="blue">
          <Group justify="space-between" align="center">
            <div>
              <Text size="sm" fw={600}>Gewichtete Gesamtbewertung</Text>
              <Text size="xs" c="dimmed">Basierend auf den gemittelten Reviewer-Bewertungen</Text>
            </div>
            <Badge size="xl" color="blue">
              {overallAverage.name} ({overallAverage.number.toFixed(2)})
            </Badge>
          </Group>
        </Alert>

        <Tabs value={activeCategory} onChange={(value) => setActiveCategory(value || '0')}>
          <Tabs.List>
            {sortedCategories.map((category, index) => (
              <Tabs.Tab 
                key={category.id} 
                value={index.toString()}
                rightSection={
                  isCategoryApproved(category.id) ? (
                    <IconCheck size={16} color="green" />
                  ) : null
                }
              >
                {category.name}
              </Tabs.Tab>
            ))}
            
            {/* Abschluss tab - only show if all categories are approved */}
            {data.all_categories_approved && (
              <Tabs.Tab 
                value="final" 
                rightSection={
                  data.final_consolidation?.is_fully_approved ? (
                    <IconCheck size={16} color="green" />
                  ) : null
                }
              >
                Abschluss
              </Tabs.Tab>
            )}
          </Tabs.List>

          {sortedCategories.map((category, index) => {
            const userResponse = data.user_responses.find(r => r.category_id === category.id);
            const averagedResponse = data.averaged_responses.find(r => r.category_id === category.id);
            const currentUserReview = data.current_user_responses?.find(r => r.category_id === category.id);
            const override = overrideData[category.id];
            const existingOverride = data.overrides?.find(o => o.category_id === category.id);
            const categoryPaths = category.paths || [];

            return (
              <Tabs.Panel key={category.id} value={index.toString()} pt="lg">
                <Grid>
                  <Grid.Col span={{ base: 12, md: 6 }}>
                    <Stack gap="md">
                      <Paper p="md" withBorder>
                        <Title order={4} mb="md">Benutzer-Einschätzung</Title>
                        {userResponse ? (
                          <Stack gap="sm">
                            <div>
                              <Text size="sm" fw={600} c="dimmed">Pfad</Text>
                              <Text>{userResponse.path_name}</Text>
                            </div>
                            <div>
                              <Text size="sm" fw={600} c="dimmed">Stufe</Text>
                              <Badge>{userResponse.level_name} (Stufe {userResponse.level_number})</Badge>
                            </div>
                            <div>
                              <Text size="sm" fw={600} c="dimmed">Begründung</Text>
                              <Paper p="sm" withBorder bg={colorScheme === 'dark' ? 'dark.6' : 'gray.0'}>
                                <Text size="sm">{userResponse.justification || 'Keine Begründung'}</Text>
                              </Paper>
                            </div>
                          </Stack>
                        ) : (
                          <Text c="dimmed">Keine Benutzer-Einschätzung vorhanden</Text>
                        )}
                      </Paper>

                      <Paper p="md" withBorder bg={colorScheme === 'dark' ? 'dark.7' : 'gray.1'}>
                        <Group justify="space-between" mb="md">
                          <Title order={4}>Ihre Reviewer-Bewertung</Title>
                          <Badge color="gray">Read-only</Badge>
                        </Group>
                        {currentUserReview ? (
                          <Stack gap="sm">
                            <div>
                              <Text size="sm" fw={600} c="dimmed">Pfad</Text>
                              <Text>{categoryPaths.find((p: any) => p.id === currentUserReview.path_id)?.name || '-'}</Text>
                            </div>
                            <div>
                              <Text size="sm" fw={600} c="dimmed">Stufe</Text>
                              <Badge>{data.catalog.levels.find((l: any) => l.id === currentUserReview.level_id)?.name} (Stufe {data.catalog.levels.find((l: any) => l.id === currentUserReview.level_id)?.level_number})</Badge>
                            </div>
                            <div>
                              <Text size="sm" fw={600} c="dimmed">Begründung</Text>
                              <Paper p="sm" withBorder bg={colorScheme === 'dark' ? 'dark.6' : 'white'}>
                                <Text size="sm">{currentUserReview.justification || 'Keine Begründung'}</Text>
                              </Paper>
                            </div>
                          </Stack>
                        ) : (
                          <Text c="dimmed">Sie haben diese Kategorie noch nicht bewertet</Text>
                        )}
                      </Paper>
                    </Stack>
                  </Grid.Col>

                  <Grid.Col span={{ base: 12, md: 6 }}>
                    <Stack gap="md">
                      <Paper p="md" withBorder>
                        <Title order={4} mb="md">Gemittelte Reviewer-Bewertung</Title>
                        {averagedResponse ? (
                          <Stack gap="sm">
                            <div>
                              <Text size="sm" fw={600} c="dimmed">Durchschnittliche Stufe</Text>
                              <Badge size="lg" color="blue">
                                {averagedResponse.average_level_name} ({averagedResponse.average_level_number.toFixed(2)})
                              </Badge>
                              <Text size="xs" c="dimmed" mt={4}>
                                Basierend auf {averagedResponse.reviewer_count} Review(s)
                              </Text>
                            </div>

                            {/* Show approval badges if averaged response has approvals */}
                            {!existingOverride && averagedResponse.approvals && averagedResponse.approvals.length > 0 && (
                              <div>
                                <Group gap="xs" mb="xs">
                                  <Text size="sm" fw={500}>Bestätigungen ({averagedResponse.approval_count}/2):</Text>
                                </Group>
                                <Group gap="xs">
                                  {averagedResponse.approvals.map((approval) => (
                                    <Tooltip key={approval.id} label={`Bestätigt am ${new Date(approval.approved_at).toLocaleDateString()}`}>
                                      <Badge 
                                        leftSection={<IconCheck size={12} />}
                                        color="green"
                                        variant="light"
                                      >
                                        {approval.approved_by_name}
                                      </Badge>
                                    </Tooltip>
                                  ))}
                                </Group>
                              </div>
                            )}

                            {/* Show approve/revoke button for averaged response (only if no override exists) */}
                            {!existingOverride && (
                              averagedResponse?.approvals?.some(a => a.approved_by_user_id === user?.id) ? (
                                canRevokeApprovals && (
                                  <Button 
                                    onClick={() => handleRevokeAveragedApproval(category.id)}
                                    color="orange"
                                    variant="light"
                                  >
                                    Bestätigung zurücknehmen
                                  </Button>
                                )
                              ) : (
                                !isReadOnly && (
                                  <Button 
                                    onClick={() => handleApproveAveraged(category.id)}
                                    color="green"
                                    leftSection={<IconCheck size={16} />}
                                    variant="light"
                                  >
                                    Gemittelte Bewertung bestätigen
                                  </Button>
                                )
                              )
                            )}
                          </Stack>
                        ) : (
                          <Text c="dimmed">Keine Reviewer-Bewertungen vorhanden</Text>
                        )}
                      </Paper>

                      <Paper p="md" withBorder>
                        <Group justify="space-between" mb="md">
                          <Title order={5}>Manuelle Anpassung</Title>
                          {existingOverride && (
                            <Badge 
                              color={isCurrentUserAuthor(existingOverride) ? "blue" : "gray"}
                              variant="light"
                            >
                              {isCurrentUserAuthor(existingOverride) ? "Ihre Anpassung" : "Anpassung vorhanden"}
                            </Badge>
                          )}
                        </Group>
                        <Stack gap="md">
                          <Select
                            label="Pfad"
                            description="Wählen Sie den Entwicklungspfad"
                            placeholder="Pfad auswählen"
                            data={categoryPaths.map((p: any) => ({ 
                              value: p.id.toString(), 
                              label: p.name 
                            }))}
                            value={override?.path_id?.toString() || ''}
                            onChange={(value) => {
                              const pathId = parseInt(value || '0');
                              setOverrideData(prev => ({
                                ...prev,
                                [category.id]: { 
                                  ...prev[category.id], 
                                  path_id: pathId,
                                  level_id: undefined // Reset level when path changes
                                }
                              }));
                            }}
                            disabled={isReadOnly}
                          />

                          <Radio.Group
                            label="Stufe auswählen"
                            description="Wählen Sie die Reifegradstufe"
                            value={override?.level_id?.toString() || ''}
                            onChange={(value) => {
                              setOverrideData(prev => ({
                                ...prev,
                                [category.id]: { 
                                  ...prev[category.id], 
                                  level_id: parseInt(value)
                                }
                              }));
                            }}
                            disabled={isReadOnly}
                          >
                            <Stack gap="sm" mt="xs">
                              {data.catalog.levels.map((level: any) => {
                                const pathLevelDesc = override?.path_id 
                                  ? categoryPaths
                                      .find((p: any) => p.id === override.path_id)
                                      ?.descriptions?.find((d: any) => d.level_id === level.id)
                                  : null;

                                return (
                                  <Paper 
                                    key={level.id} 
                                    p="sm" 
                                    withBorder
                                    style={{
                                      opacity: override?.path_id ? 1 : 0.6,
                                      backgroundColor: override?.level_id === level.id ? '#e7f5ff' : 'white'
                                    }}
                                  >
                                    <Radio
                                      value={level.id.toString()}
                                      label={
                                        <div>
                                          <Group gap="xs" mb="xs">
                                            <Text fw={600}>{level.name}</Text>
                                            <Badge size="sm" variant="light">Stufe {level.level_number}</Badge>
                                          </Group>
                                          <Text size="xs" c="dimmed">{level.description}</Text>
                                          {pathLevelDesc && (
                                            <Paper p="xs" withBorder bg={colorScheme === 'dark' ? 'dark.5' : 'blue.0'} mt="xs">
                                              <Text size="xs" fw={600} c="blue">Pfad-spezifische Beschreibung:</Text>
                                              <Text size="xs">{pathLevelDesc.description}</Text>
                                            </Paper>
                                          )}
                                        </div>
                                      }
                                      disabled={!override?.path_id}
                                    />
                                  </Paper>
                                );
                              })}
                            </Stack>
                          </Radio.Group>

                          <Textarea
                            label="Begründung"
                            description="Erklären Sie, warum Sie diese Anpassung vornehmen"
                            placeholder="Begründung für die Anpassung (erforderlich)"
                            rows={4}
                            value={override?.justification || ''}
                            onChange={(e) => {
                              const value = e.target.value;
                              setOverrideData(prev => ({
                                ...prev,
                                [category.id]: { ...prev[category.id], justification: value }
                              }));
                            }}
                            disabled={isReadOnly}
                          />
                          
                          {/* Show approval badges if override exists and has approvals */}
                          {existingOverride && existingOverride.approvals && existingOverride.approvals.length > 0 && (
                            <Group gap="xs">
                              <Text size="sm" fw={500}>Bestätigungen:</Text>
                              {existingOverride.approvals.map((approval) => (
                                <Tooltip key={approval.id} label={`Bestätigt am ${new Date(approval.approved_at).toLocaleDateString()}`}>
                                  <Badge 
                                    leftSection={<IconCheck size={12} />}
                                    color="green"
                                    variant="light"
                                  >
                                    {approval.approved_by_name}
                                  </Badge>
                                </Tooltip>
                              ))}
                            </Group>
                          )}
                          
                          {/* Conditional buttons */}
                          <Group>
                            {existingOverride && !isCurrentUserAuthor(existingOverride) && !hasOverrideChanged(category.id, existingOverride) ? (
                              // User can approve or revoke their approval
                              existingOverride.approvals?.some(a => a.approved_by_user_id === user?.id) ? (
                                canRevokeApprovals && (
                                  <Button 
                                    onClick={() => handleRevokeOverrideApproval(category.id)}
                                    color="orange"
                                    variant="light"
                                    flex={1}
                                  >
                                    Bestätigung zurücknehmen
                                  </Button>
                                )
                              ) : (
                                !isReadOnly && (
                                  <Button 
                                    onClick={() => handleApproveOverride(category.id)}
                                    color="green"
                                    leftSection={<IconCheck size={16} />}
                                    flex={1}
                                  >
                                    Anpassung bestätigen
                                  </Button>
                                )
                              )
                            ) : (
                              !isReadOnly && (
                                <Button 
                                  onClick={() => handleSaveOverride(category.id)}
                                  disabled={!override?.path_id || !override?.level_id || !override?.justification}
                                  flex={1}
                                >
                                  Anpassung speichern
                                </Button>
                              )
                            )}
                            
                            {/* Show delete button if override exists */}
                            {existingOverride && !isReadOnly && (
                              <Button 
                                onClick={() => handleDeleteOverride(category.id)}
                                color="red"
                                variant="light"
                              >
                                Löschen
                              </Button>
                            )}
                          </Group>
                        </Stack>
                      </Paper>
                    </Stack>
                  </Grid.Col>
                </Grid>
              </Tabs.Panel>
            );
          })}

          {/* Abschluss Tab Panel */}
          {data.all_categories_approved && (
            <Tabs.Panel value="final" pt="lg">
              <Stack gap="lg">
                <Alert color="blue" title="Abschluss der Konsolidierung">
                  <Text size="sm">
                    Alle Kategorien wurden genehmigt. Bitte verfassen Sie einen abschließenden Kommentar zur Gesamtbewertung. 
                    Nach Zustimmung aller Reviewer wird die Bewertung als abgeschlossen markiert.
                  </Text>
                </Alert>

                {/* Category Results Summary with Comments */}
                <Paper withBorder p="md">
                  <Stack gap="lg">
                    <Title order={4}>Kategorie-Ergebnisse und Kommentare für Besprechungs-Ansicht</Title>
                    {sortedCategories.map(category => {
                      const override = data.overrides?.find(o => o.category_id === category.id);
                      const averaged = data.averaged_responses.find(r => r.category_id === category.id);
                      
                      let resultLevel = null;
                      let isOverride = false;
                      
                      if (override && override.is_approved) {
                        // Find the level for the override from catalog levels
                        const level = data.catalog.levels?.find((l: any) => l.id === override.level_id);
                        if (level) {
                          resultLevel = {
                            level_number: level.level_number,
                            name: level.name
                          };
                        }
                        isOverride = true;
                      } else if (averaged && averaged.is_approved) {
                        resultLevel = {
                          level_number: averaged.average_level_number,
                          name: averaged.average_level_name
                        };
                      }

                      return (
                        <Paper key={category.id} withBorder p="md" bg={colorScheme === 'dark' ? 'dark.6' : 'gray.0'}>
                          <Stack gap="md">
                            <Group justify="space-between">
                              <div>
                                <Text size="md" fw={600}>{category.name}</Text>
                                <Text size="xs" c="dimmed">Gewicht: {category.weight}</Text>
                              </div>
                              {resultLevel && (
                                <Badge size="lg" color={isOverride ? 'blue' : 'green'}>
                                  {resultLevel.name} ({resultLevel.level_number})
                                </Badge>
                              )}
                            </Group>
                            
                            <div>
                              <Text size="sm" fw={600} mb="xs">Kommentar für Besprechungs-Ansicht</Text>
                              <Text size="xs" c="dimmed" mb="sm">
                                Dieser Kommentar wird dem Mitarbeiter in der Besprechungs-Ansicht angezeigt.
                              </Text>
                              <Textarea
                                placeholder="Erklären Sie die Bewertung für diese Kategorie..."
                                rows={4}
                                value={categoryComments[category.id] || ''}
                                onChange={(e) => {
                                  const value = e.target.value;
                                  setCategoryComments(prev => ({
                                    ...prev,
                                    [category.id]: value
                                  }));
                                }}
                                disabled={isReadOnly}
                              />
                              {!isReadOnly && (
                                <Button 
                                  onClick={() => handleSaveCategoryComment(category.id)}
                                  disabled={!categoryComments[category.id]?.trim()}
                                  mt="sm"
                                  size="sm"
                                >
                                  Kommentar speichern
                                </Button>
                              )}
                            </div>
                          </Stack>
                        </Paper>
                      );
                    })}
                  </Stack>
                </Paper>

                {/* Overall Average */}
                <Paper withBorder p="md" bg={colorScheme === 'dark' ? 'dark.5' : 'blue.0'}>
                  <Group justify="space-between">
                    <div>
                      <Text size="lg" fw={700}>Gesamtbewertung</Text>
                      <Text size="sm" c="dimmed">Gewichteter Durchschnitt aller Kategorien</Text>
                    </div>
                    <Badge size="xl" color="blue">
                      {overallAverage.name} ({overallAverage.number.toFixed(2)})
                    </Badge>
                  </Group>
                </Paper>

                {/* Final Comment */}
                <Paper withBorder p="md">
                  <Stack gap="md">
                    <div>
                      <Title order={4}>Abschluss-Kommentar</Title>
                      <Text size="sm" c="dimmed">
                        Verfassen Sie einen zusammenfassenden Kommentar zur Gesamtbewertung
                      </Text>
                    </div>
                    <Textarea
                      value={finalComment}
                      onChange={(e) => setFinalComment(e.target.value)}
                      placeholder="ToDo: LLM-Zusammenfassung - Geben Sie hier einen abschließenden Kommentar zur Konsolidierung ein..."
                      minRows={8}
                      required
                      disabled={isReadOnly}
                    />
                    
                    {/* Approvals */}
                    {data.final_consolidation && (
                      <div>
                        <Text size="sm" fw={600} mb="xs">
                          Bestätigungen ({data.final_consolidation.approval_count || 0} von {data.final_consolidation.required_approvals || 0})
                        </Text>
                        {data.final_consolidation.approvals && data.final_consolidation.approvals.length > 0 ? (
                          <Group gap="xs">
                            {data.final_consolidation.approvals.map(approval => (
                              <Badge key={approval.id} color="green" leftSection={<IconCheck size={12} />}>
                                {approval.approved_by_name}
                              </Badge>
                            ))}
                          </Group>
                        ) : (
                          <Text size="sm" c="dimmed">Noch keine Bestätigungen</Text>
                        )}
                      </div>
                    )}

                    <Group>
                      {!isReadOnly && (
                        <Button onClick={handleSaveFinalComment} disabled={!finalComment.trim()}>
                          Kommentar speichern
                        </Button>
                      )}
                      
                      {data.final_consolidation && (
                        data.final_consolidation.approvals?.some(a => a.approved_by_user_id === user?.id) ? (
                          canRevokeApprovals && (
                            <Button 
                              onClick={handleRevokeFinalApproval}
                              color="orange"
                              variant="light"
                            >
                              Bestätigung zurücknehmen
                            </Button>
                          )
                        ) : (
                          !isReadOnly && (
                            <Button 
                              onClick={handleApproveFinal}
                              color="green"
                              leftSection={<IconCheck size={16} />}
                              disabled={!finalComment.trim()}
                            >
                              Abschluss bestätigen
                            </Button>
                          )
                        )
                      )}
                    </Group>

                    {data.final_consolidation?.is_fully_approved && (
                      <Alert color="green" title="Konsolidierung abgeschlossen">
                        Alle Reviewer haben den Abschluss bestätigt. Die Bewertung wurde als "reviewed" markiert 
                        und der/die Benutzer:in wurde benachrichtigt.
                      </Alert>
                    )}
                  </Stack>
                </Paper>
              </Stack>
            </Tabs.Panel>
          )}
        </Tabs>
      </Stack>
    </Container>
  );
}
