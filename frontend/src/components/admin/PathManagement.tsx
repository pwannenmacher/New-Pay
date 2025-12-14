import { useState, useEffect } from 'react';
import {
  Stack,
  Group,
  Button,
  ActionIcon,
  Table,
  Modal,
  TextInput,
  Textarea,
  Accordion,
  Title,
  Text,
  Alert,
  Progress,
} from '@mantine/core';
import { IconPlus, IconEdit, IconTrash, IconArrowUp, IconArrowDown } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { adminApi } from '../../services/admin';
import type { Category, Path, Level } from '../../types';

interface PathManagementProps {
  catalogId: number;
  categories: Category[];
  levels: Level[];
  phase: string;
  onUpdate?: () => void;
}

export function PathManagement({
  catalogId,
  categories,
  levels,
  phase,
  onUpdate,
}: PathManagementProps) {
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const [paths, setPaths] = useState<Path[]>([]);
  const [pathModalOpened, setPathModalOpened] = useState(false);
  const [editingPath, setEditingPath] = useState<Path | null>(null);
  const [newPathName, setNewPathName] = useState('');
  const [newPathDescription, setNewPathDescription] = useState('');
  const [categoryPathCounts, setCategoryPathCounts] = useState<Record<number, number>>({});

  const [descriptionModalOpened, setDescriptionModalOpened] = useState(false);
  const [editingPathForDescriptions, setEditingPathForDescriptions] = useState<Path | null>(null);
  const [descriptionTexts, setDescriptionTexts] = useState<Record<number, string>>({});

  const loadAllPathCounts = async () => {
    try {
      const catalog = await adminApi.getCatalog(catalogId);
      const counts: Record<number, number> = {};
      catalog.categories?.forEach((cat) => {
        counts[cat.id] = cat.paths?.length || 0;
      });
      setCategoryPathCounts(counts);
    } catch (err: any) {
      // Silent fail for counts
    }
  };

  useEffect(() => {
    if (categories.length > 0) {
      loadAllPathCounts();
    }
  }, [catalogId, categories.length]);

  const loadPathsForCategory = async (category: Category) => {
    try {
      const catalog = await adminApi.getCatalog(catalogId);
      const cat = catalog.categories?.find((c) => c.id === category.id);
      setPaths(cat?.paths || []);
      setSelectedCategory(category);
      loadAllPathCounts(); // Refresh counts
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: 'Pfade konnten nicht geladen werden',
        color: 'red',
      });
    }
  };

  const handleOpenPathModal = () => {
    if (!selectedCategory) return;
    setEditingPath(null);
    setNewPathName(`Pfad ${paths.length + 1}`);
    setNewPathDescription('');
    setPathModalOpened(true);
  };

  const handleOpenEditPathModal = (path: Path) => {
    setEditingPath(path);
    setNewPathName(path.name);
    setNewPathDescription(path.description || '');
    setPathModalOpened(true);
  };

  const handleSavePath = async () => {
    if (!selectedCategory || !newPathName.trim()) return;

    try {
      if (editingPath) {
        const updatedPath = await adminApi.updatePath(
          catalogId,
          selectedCategory.id,
          editingPath.id,
          {
            name: newPathName.trim(),
            description: newPathDescription.trim() || undefined,
            sort_order: editingPath.sort_order,
          }
        );
        setPaths(paths.map((p) => (p.id === editingPath.id ? updatedPath : p)));
        notifications.show({
          title: 'Erfolg',
          message: 'Pfad erfolgreich aktualisiert',
          color: 'green',
        });
      } else {
        const newPath = await adminApi.createPath(catalogId, selectedCategory.id, {
          name: newPathName.trim(),
          description: newPathDescription.trim() || undefined,
          sort_order: paths.length,
        });
        setPaths([...paths, newPath]);
        notifications.show({
          title: 'Erfolg',
          message: 'Pfad erfolgreich hinzugefügt',
          color: 'green',
        });

        // Refresh catalog data to update progress
        if (onUpdate) {
          onUpdate();
        }
      }
      setPathModalOpened(false);
      setEditingPath(null);
      setNewPathName('');
      setNewPathDescription('');
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Speichern',
        color: 'red',
      });
    }
  };

  const handleMovePath = async (index: number, direction: 'up' | 'down') => {
    if (!selectedCategory) return;
    if ((direction === 'up' && index === 0) || (direction === 'down' && index === paths.length - 1))
      return;

    const newPaths = [...paths];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    [newPaths[index], newPaths[targetIndex]] = [newPaths[targetIndex], newPaths[index]];

    try {
      // Temporary high numbers
      for (let i = 0; i < newPaths.length; i++) {
        await adminApi.updatePath(catalogId, selectedCategory.id, newPaths[i].id, {
          name: newPaths[i].name,
          description: newPaths[i].description,
          sort_order: 1000 + i,
        });
      }

      // Final numbers
      const updatedPaths = [];
      for (let i = 0; i < newPaths.length; i++) {
        const updated = await adminApi.updatePath(catalogId, selectedCategory.id, newPaths[i].id, {
          name: newPaths[i].name,
          description: newPaths[i].description,
          sort_order: i,
        });
        updatedPaths.push(updated);
      }

      setPaths(updatedPaths);
      notifications.show({
        title: 'Erfolg',
        message: 'Reihenfolge erfolgreich geändert',
        color: 'green',
      });
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Umsortieren',
        color: 'red',
      });
      if (selectedCategory) loadPathsForCategory(selectedCategory);
    }
  };

  const handleDeletePath = async (pathId: number) => {
    if (!selectedCategory || !confirm('Pfad wirklich löschen?')) return;

    try {
      await adminApi.deletePath(catalogId, selectedCategory.id, pathId);
      const remainingPaths = paths.filter((p) => p.id !== pathId);

      const updatedPaths = remainingPaths.map((path, index) => ({
        ...path,
        sort_order: index,
      }));

      await Promise.all(
        updatedPaths.map((path) =>
          adminApi.updatePath(catalogId, selectedCategory.id, path.id, {
            name: path.name,
            description: path.description,
            sort_order: path.sort_order,
          })
        )
      );

      setPaths(updatedPaths);
      notifications.show({
        title: 'Erfolg',
        message: 'Pfad erfolgreich gelöscht',
        color: 'green',
      });

      // Refresh catalog data to update progress
      if (onUpdate) {
        onUpdate();
      }
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Löschen',
        color: 'red',
      });
    }
  };

  const handleOpenDescriptionsModal = async (path: Path) => {
    setEditingPathForDescriptions(path);

    try {
      const catalog = await adminApi.getCatalog(catalogId);
      const cat = catalog.categories?.find((c) => c.id === selectedCategory?.id);
      const pathWithDescriptions = cat?.paths?.find((p) => p.id === path.id);

      // Initialize description texts
      const texts: Record<number, string> = {};
      levels.forEach((level) => {
        const desc = pathWithDescriptions?.descriptions?.find((d) => d.level_id === level.id);
        texts[level.id] = desc?.description || '';
      });
      setDescriptionTexts(texts);
      setDescriptionModalOpened(true);
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: 'Beschreibungen konnten nicht geladen werden',
        color: 'red',
      });
    }
  };

  const handleSaveDescriptions = async () => {
    if (!editingPathForDescriptions) return;

    try {
      for (const level of levels) {
        const description = descriptionTexts[level.id];
        if (description && description.trim()) {
          await adminApi.saveDescription(catalogId, {
            path_id: editingPathForDescriptions.id,
            level_id: level.id,
            description: description.trim(),
          });
        }
      }

      setDescriptionModalOpened(false);
      setEditingPathForDescriptions(null);
      setDescriptionTexts({});

      notifications.show({
        title: 'Erfolg',
        message: 'Beschreibungen erfolgreich gespeichert',
        color: 'green',
      });

      // Refresh catalog data to update progress
      if (onUpdate) {
        onUpdate();
      }
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Speichern',
        color: 'red',
      });
    }
  };

  const calculatePathProgress = (path: Path) => {
    const totalDescriptions = levels.length;
    const filledDescriptions =
      (path as any).descriptions?.filter((d: any) => d.description && d.description.trim())
        .length || 0;
    const percentage =
      totalDescriptions > 0 ? Math.round((filledDescriptions / totalDescriptions) * 100) : 0;
    return { filledDescriptions, totalDescriptions, percentage };
  };

  if (levels.length === 0) {
    return (
      <Alert color="blue">
        Bitte erstellen Sie zuerst Levels im "Levels"-Tab, bevor Sie Pfade anlegen.
      </Alert>
    );
  }

  return (
    <Stack gap="md">
      <Accordion>
        {categories.map((category) => (
          <Accordion.Item key={category.id} value={category.id.toString()}>
            <Accordion.Control onClick={() => loadPathsForCategory(category)}>
              <Group justify="space-between">
                <Text fw={500}>{category.name}</Text>
                <Text size="sm" c="dimmed">
                  {categoryPathCounts[category.id] ?? 0} Pfad(e)
                </Text>
              </Group>
            </Accordion.Control>
            <Accordion.Panel>
              {selectedCategory?.id === category.id && (
                <Stack gap="md">
                  <Group justify="space-between">
                    <Title order={5}>Pfade in "{category.name}"</Title>
                    <Button
                      leftSection={<IconPlus size={16} />}
                      onClick={handleOpenPathModal}
                      size="sm"
                      disabled={phase !== 'draft'}
                    >
                      Pfad hinzufügen
                    </Button>
                  </Group>

                  {paths.length === 0 ? (
                    <Alert color="blue">Noch keine Pfade in dieser Kategorie</Alert>
                  ) : (
                    <Table>
                      <Table.Thead>
                        <Table.Tr>
                          <Table.Th>Name</Table.Th>
                          <Table.Th>Beschreibung</Table.Th>
                          <Table.Th>Fortschritt</Table.Th>
                          <Table.Th>Sortierung</Table.Th>
                          <Table.Th>Aktionen</Table.Th>
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {paths.map((path, index) => {
                          const progress = calculatePathProgress(path);
                          return (
                            <Table.Tr key={path.id}>
                              <Table.Td>{path.name}</Table.Td>
                              <Table.Td>{path.description || '-'}</Table.Td>
                              <Table.Td>
                                <Stack gap={4}>
                                  <Progress
                                    value={progress.percentage}
                                    color={progress.percentage === 100 ? 'green' : 'blue'}
                                    size="sm"
                                  />
                                  <Text size="xs" c="dimmed">
                                    {progress.filledDescriptions}/{progress.totalDescriptions} (
                                    {progress.percentage}%)
                                  </Text>
                                </Stack>
                              </Table.Td>
                              <Table.Td>{path.sort_order}</Table.Td>
                              <Table.Td>
                                <Group gap="xs">
                                  <ActionIcon
                                    variant="light"
                                    color="gray"
                                    onClick={() => handleMovePath(index, 'up')}
                                    disabled={index === 0 || phase !== 'draft'}
                                  >
                                    <IconArrowUp size={16} />
                                  </ActionIcon>
                                  <ActionIcon
                                    variant="light"
                                    color="gray"
                                    onClick={() => handleMovePath(index, 'down')}
                                    disabled={index === paths.length - 1 || phase !== 'draft'}
                                  >
                                    <IconArrowDown size={16} />
                                  </ActionIcon>
                                  <Button
                                    variant="light"
                                    color="green"
                                    size="xs"
                                    onClick={() => handleOpenDescriptionsModal(path)}
                                  >
                                    Level-Beschreibungen
                                  </Button>
                                  <ActionIcon
                                    variant="light"
                                    color="blue"
                                    onClick={() => handleOpenEditPathModal(path)}
                                    disabled={phase !== 'draft'}
                                  >
                                    <IconEdit size={16} />
                                  </ActionIcon>
                                  <ActionIcon
                                    variant="light"
                                    color="red"
                                    onClick={() => handleDeletePath(path.id)}
                                    disabled={phase !== 'draft'}
                                  >
                                    <IconTrash size={16} />
                                  </ActionIcon>
                                </Group>
                              </Table.Td>
                            </Table.Tr>
                          );
                        })}
                      </Table.Tbody>
                    </Table>
                  )}
                </Stack>
              )}
            </Accordion.Panel>
          </Accordion.Item>
        ))}
      </Accordion>

      {/* Path Modal */}
      <Modal
        opened={pathModalOpened}
        onClose={() => {
          setPathModalOpened(false);
          setEditingPath(null);
          setNewPathName('');
          setNewPathDescription('');
        }}
        title={editingPath ? 'Pfad bearbeiten' : 'Neuen Pfad hinzufügen'}
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (newPathName.trim()) {
              handleSavePath();
            }
          }}
        >
          <Stack gap="md">
            <TextInput
              label="Pfad-Name"
              placeholder="z.B. Technische Fähigkeiten"
              value={newPathName}
              onChange={(e) => setNewPathName(e.target.value)}
              required
              data-autofocus
            />
            <Textarea
              label="Beschreibung"
              placeholder="Optionale Beschreibung des Pfads"
              value={newPathDescription}
              onChange={(e) => setNewPathDescription(e.target.value)}
              rows={3}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault();
                  if (newPathName.trim()) {
                    handleSavePath();
                  }
                }
              }}
            />
            <Group justify="flex-end">
              <Button
                variant="subtle"
                onClick={() => {
                  setPathModalOpened(false);
                  setEditingPath(null);
                  setNewPathName('');
                  setNewPathDescription('');
                }}
                type="button"
              >
                Abbrechen
              </Button>
              <Button type="submit" disabled={!newPathName.trim()}>
                {editingPath ? 'Speichern' : 'Hinzufügen'}
              </Button>
            </Group>
          </Stack>
        </form>
      </Modal>

      {/* Level Descriptions Modal */}
      <Modal
        opened={descriptionModalOpened}
        onClose={() => {
          setDescriptionModalOpened(false);
          setEditingPathForDescriptions(null);
          setDescriptionTexts({});
        }}
        title={`Level-Beschreibungen für "${editingPathForDescriptions?.name}"`}
        size="lg"
      >
        <Stack gap="md">
          {levels.map((level) => (
            <Textarea
              key={level.id}
              label={`${level.name} (Level ${level.level_number})`}
              placeholder={`Beschreibung für ${level.name}...`}
              value={descriptionTexts[level.id] || ''}
              onChange={(e) =>
                setDescriptionTexts({
                  ...descriptionTexts,
                  [level.id]: e.target.value,
                })
              }
              rows={3}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault();
                  handleSaveDescriptions();
                }
              }}
            />
          ))}
          <Group justify="flex-end">
            <Button
              variant="subtle"
              onClick={() => {
                setDescriptionModalOpened(false);
                setEditingPathForDescriptions(null);
                setDescriptionTexts({});
              }}
            >
              Abbrechen
            </Button>
            <Button onClick={handleSaveDescriptions}>Alle Beschreibungen speichern</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
