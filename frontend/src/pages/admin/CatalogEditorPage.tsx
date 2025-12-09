import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Container,
  Title,
  Paper,
  Stack,
  TextInput,
  Textarea,
  Button,
  Group,
  Loader,
  Alert,
  Tabs,
  Table,
  ActionIcon,
  Badge,
  Modal,
  Select,
  Progress,
  Text,
} from '@mantine/core';
import {
  IconAlertCircle,
  IconDeviceFloppy,
  IconArrowLeft,
  IconPlus,
  IconTrash,
  IconEdit,
  IconArrowUp,
  IconArrowDown,
} from '@tabler/icons-react';
import { DatePickerInput } from '@mantine/dates';
import { notifications } from '@mantine/notifications';
import { adminApi } from '../../services/admin';
import { PathManagement } from '../../components/admin/PathManagement';
import type {
  CatalogWithDetails,
  Category,
  CategoryWithPaths,
  Level,
} from '../../types';

export function CatalogEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isNew = id === 'new';

  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Catalog fields
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [validFrom, setValidFrom] = useState<Date | null>(null);
  const [validUntil, setValidUntil] = useState<Date | null>(null);
  const [phase, setPhase] = useState<string>('draft');

  // Nested entities
  const [catalog, setCatalog] = useState<CatalogWithDetails | null>(null);
  const [categories, setCategories] = useState<CategoryWithPaths[]>([]);
  const [levels, setLevels] = useState<Level[]>([]);

  // Active tab
  const [activeTab, setActiveTab] = useState<string | null>('basic');

  // Level modal
  const [levelModalOpened, setLevelModalOpened] = useState(false);
  const [editingLevel, setEditingLevel] = useState<Level | null>(null);
  const [newLevelName, setNewLevelName] = useState('');
  const [newLevelDescription, setNewLevelDescription] = useState('');

  // Category modal
  const [categoryModalOpened, setCategoryModalOpened] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [newCategoryName, setNewCategoryName] = useState('');
  const [newCategoryDescription, setNewCategoryDescription] = useState('');

  useEffect(() => {
    if (!isNew && id) {
      const catalogId = parseInt(id, 10);
      // Only load if we have a valid numeric ID
      if (!isNaN(catalogId) && catalogId > 0) {
        loadCatalog(catalogId);
      }
    }
  }, [id, isNew]);

  const loadCatalog = async (catalogId: number) => {
    try {
      setLoading(true);
      setError(null);
      const data = await adminApi.getCatalog(catalogId);
      setCatalog(data);
      setName(data.name);
      setDescription(data.description || '');
      setValidFrom(new Date(data.valid_from));
      setValidUntil(new Date(data.valid_until));
      setPhase(data.phase || 'draft');
      setCategories(data.categories?.map((c) => c) || []);
      setLevels(data.levels || []);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load catalog');
    } finally {
      setLoading(false);
    }
  };

  const calculateCompletionStats = () => {
    const totalPaths = categories.reduce((sum, cat) => sum + (cat.paths?.length || 0), 0);
    const totalDescriptionsNeeded = totalPaths * levels.length;
    
    let filledDescriptions = 0;
    categories.forEach(cat => {
      cat.paths?.forEach(path => {
        path.descriptions?.forEach(desc => {
          if (desc.description && desc.description.trim()) {
            filledDescriptions++;
          }
        });
      });
    });

    const allCategoriesHavePaths = categories.length > 0 && categories.every(cat => (cat.paths?.length || 0) > 0);
    const percentage = totalDescriptionsNeeded > 0 ? Math.round((filledDescriptions / totalDescriptionsNeeded) * 100) : 0;
    
    return {
      totalPaths,
      totalDescriptionsNeeded,
      filledDescriptions,
      percentage,
      allCategoriesHavePaths,
    };
  };

  const handleSaveBasic = async () => {
    // Validate required fields
    const errors: string[] = [];
    if (!name.trim()) errors.push('Name');
    if (!validFrom) errors.push('Gültig von');
    if (!validUntil) errors.push('Gültig bis');
    
    if (errors.length > 0) {
      notifications.show({
        title: 'Fehler',
        message: `Bitte füllen Sie folgende Pflichtfelder aus: ${errors.join(', ')}`,
        color: 'red',
      });
      return;
    }

    setSaving(true);
    try {
      // TypeScript doesn't know these are non-null after validation, so we assert
      if (!validFrom || !validUntil) return; // Double check for TypeScript
      
      // Ensure we have Date objects
      const fromDate = validFrom instanceof Date ? validFrom : new Date(validFrom);
      const untilDate = validUntil instanceof Date ? validUntil : new Date(validUntil);
      
      const data = {
        name,
        description: description || undefined,
        valid_from: fromDate.toISOString().split('T')[0],
        valid_until: untilDate.toISOString().split('T')[0],
        phase: phase as any,
      };

      if (isNew) {
        const result = await adminApi.createCatalog(data);
        // Update local state with created catalog
        setCatalog({
          ...result,
          categories: [],
          levels: [],
        });
        setName(result.name);
        setDescription(result.description || '');
        setValidFrom(new Date(result.valid_from));
        setValidUntil(new Date(result.valid_until));
        
        notifications.show({
          title: 'Erfolg',
          message: 'Katalog erfolgreich erstellt',
          color: 'green',
        });
        
        // Navigate to edit page without reloading
        navigate(`/admin/catalogs/${result.id}/edit`, { replace: true });
      } else if (id) {
        await adminApi.updateCatalog(parseInt(id), data);
        notifications.show({
          title: 'Erfolg',
          message: 'Katalog erfolgreich aktualisiert',
          color: 'green',
        });
        await loadCatalog(parseInt(id));
      }
    } catch (err: any) {
      console.error('Error saving catalog:', err);
      console.error('Error response:', err.response);
      console.error('Error data:', err.response?.data);
      
      const errorMessage = err.response?.data?.error || err.response?.data?.message || err.message || 'Fehler beim Speichern';
      
      notifications.show({
        title: 'Fehler beim Speichern',
        message: errorMessage,
        color: 'red',
      });
    } finally {
      setSaving(false);
    }
  };

  const handleOpenLevelModal = () => {
    const levelNumber = levels.length + 1;
    setEditingLevel(null);
    setNewLevelName(`Level ${levelNumber}`);
    setNewLevelDescription('');
    setLevelModalOpened(true);
  };

  const handleOpenEditLevelModal = (level: Level) => {
    setEditingLevel(level);
    setNewLevelName(level.name);
    setNewLevelDescription(level.description || '');
    setLevelModalOpened(true);
  };

  const handleSaveLevel = async () => {
    if (!catalog || !newLevelName.trim()) return;

    try {
      if (editingLevel) {
        // Update existing level
        const updatedLevel = await adminApi.updateLevel(catalog.id, editingLevel.id, {
          name: newLevelName.trim(),
          level_number: editingLevel.level_number, // Muss mitgesendet werden
          description: newLevelDescription.trim() || undefined,
        });
        setLevels(levels.map(l => l.id === editingLevel.id ? updatedLevel : l));
        notifications.show({
          title: 'Erfolg',
          message: 'Level erfolgreich aktualisiert',
          color: 'green',
        });
      } else {
        // Create new level
        const levelNumber = levels.length + 1;
        const newLevel = await adminApi.createLevel(catalog.id, {
          name: newLevelName.trim(),
          level_number: levelNumber,
          description: newLevelDescription.trim() || undefined,
        });
        setLevels([...levels, newLevel]);
        notifications.show({
          title: 'Erfolg',
          message: 'Level erfolgreich hinzugefügt',
          color: 'green',
        });
        
        // Reload catalog to update progress
        await loadCatalog(catalog.id);
      }
      setLevelModalOpened(false);
      setEditingLevel(null);
      setNewLevelName('');
      setNewLevelDescription('');
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Speichern',
        color: 'red',
      });
    }
  };

  const handleMoveLevel = async (index: number, direction: 'up' | 'down') => {
    if (!catalog) return;
    if ((direction === 'up' && index === 0) || (direction === 'down' && index === levels.length - 1)) return;

    const newLevels = [...levels];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    
    // Swap positions
    [newLevels[index], newLevels[targetIndex]] = [newLevels[targetIndex], newLevels[index]];
    
    try {
      // Step 1: Set all levels to temporary high numbers (1000+) to avoid conflicts
      for (let i = 0; i < newLevels.length; i++) {
        await adminApi.updateLevel(catalog.id, newLevels[i].id, {
          name: newLevels[i].name,
          level_number: 1000 + i,
          description: newLevels[i].description,
        });
      }
      
      // Step 2: Set correct final numbers
      const updatedLevels = [];
      for (let i = 0; i < newLevels.length; i++) {
        const updated = await adminApi.updateLevel(catalog.id, newLevels[i].id, {
          name: newLevels[i].name,
          level_number: i + 1,
          description: newLevels[i].description,
        });
        updatedLevels.push(updated);
      }
      
      setLevels(updatedLevels);
      
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
      // Reload to get correct state
      if (id) loadCatalog(parseInt(id));
    }
  };

  const handleDeleteLevel = async (levelId: number) => {
    if (!catalog) return;
    if (!confirm('Level wirklich löschen?')) return;

    try {
      await adminApi.deleteLevel(catalog.id, levelId);
      const remainingLevels = levels.filter((l) => l.id !== levelId);
      
      // Renumber remaining levels
      const updatedLevels = remainingLevels.map((level, index) => ({
        ...level,
        level_number: index + 1,
      }));
      
      // Update all levels in backend to fix numbering
      await Promise.all(
        updatedLevels.map(level =>
          adminApi.updateLevel(catalog.id, level.id, {
            name: level.name,
            level_number: level.level_number,
            description: level.description,
          })
        )
      );
      
      setLevels(updatedLevels);
      notifications.show({
        title: 'Erfolg',
        message: 'Level erfolgreich gelöscht',
        color: 'green',
      });
      
      // Reload catalog to update progress
      await loadCatalog(catalog.id);
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Löschen',
        color: 'red',
      });
    }
  };

  const handleOpenCategoryModal = () => {
    setEditingCategory(null);
    setNewCategoryName(`Kategorie ${categories.length + 1}`);
    setNewCategoryDescription('');
    setCategoryModalOpened(true);
  };

  const handleOpenEditCategoryModal = (category: Category) => {
    setEditingCategory(category);
    setNewCategoryName(category.name);
    setNewCategoryDescription(category.description || '');
    setCategoryModalOpened(true);
  };

  const handleSaveCategory = async () => {
    if (!catalog || !newCategoryName.trim()) return;

    try {
      if (editingCategory) {
        // Update existing category
        const updatedCategory = await adminApi.updateCategory(catalog.id, editingCategory.id, {
          name: newCategoryName.trim(),
          description: newCategoryDescription.trim() || undefined,
          sort_order: editingCategory.sort_order,
        });
        setCategories(categories.map(c => c.id === editingCategory.id ? updatedCategory : c));
        notifications.show({
          title: 'Erfolg',
          message: 'Kategorie erfolgreich aktualisiert',
          color: 'green',
        });
      } else {
        // Create new category
        const newCategory = await adminApi.createCategory(catalog.id, {
          name: newCategoryName.trim(),
          description: newCategoryDescription.trim() || undefined,
          sort_order: categories.length,
        });
        setCategories([...categories, newCategory]);
        notifications.show({
          title: 'Erfolg',
          message: 'Kategorie erfolgreich hinzugefügt',
          color: 'green',
        });
      }
      setCategoryModalOpened(false);
      setEditingCategory(null);
      setNewCategoryName('');
      setNewCategoryDescription('');
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Speichern',
        color: 'red',
      });
    }
  };

  const handleMoveCategory = async (index: number, direction: 'up' | 'down') => {
    if (!catalog) return;
    if ((direction === 'up' && index === 0) || (direction === 'down' && index === categories.length - 1)) return;

    const newCategories = [...categories];
    const targetIndex = direction === 'up' ? index - 1 : index + 1;
    
    // Swap positions
    [newCategories[index], newCategories[targetIndex]] = [newCategories[targetIndex], newCategories[index]];
    
    try {
      // Step 1: Set temporary high numbers
      for (let i = 0; i < newCategories.length; i++) {
        await adminApi.updateCategory(catalog.id, newCategories[i].id, {
          name: newCategories[i].name,
          description: newCategories[i].description,
          sort_order: 1000 + i,
        });
      }
      
      // Step 2: Set correct final numbers
      const updatedCategories = [];
      for (let i = 0; i < newCategories.length; i++) {
        const updated = await adminApi.updateCategory(catalog.id, newCategories[i].id, {
          name: newCategories[i].name,
          description: newCategories[i].description,
          sort_order: i,
        });
        updatedCategories.push(updated);
      }
      
      setCategories(updatedCategories);
      
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
      if (id) loadCatalog(parseInt(id));
    }
  };

  const handleDeleteCategory = async (categoryId: number) => {
    if (!catalog) return;
    if (!confirm('Kategorie wirklich löschen?')) return;

    try {
      await adminApi.deleteCategory(catalog.id, categoryId);
      const remainingCategories = categories.filter((c) => c.id !== categoryId);
      
      // Renumber remaining categories
      const updatedCategories = remainingCategories.map((category, index) => ({
        ...category,
        sort_order: index,
      }));
      
      // Update all categories in backend
      await Promise.all(
        updatedCategories.map(category =>
          adminApi.updateCategory(catalog.id, category.id, {
            name: category.name,
            description: category.description,
            sort_order: category.sort_order,
          })
        )
      );
      
      setCategories(updatedCategories);
      notifications.show({
        title: 'Erfolg',
        message: 'Kategorie erfolgreich gelöscht',
        color: 'green',
      });
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Löschen',
        color: 'red',
      });
    }
  };

  if (loading) {
    return (
      <Container size="xl" py="xl">
        <Stack align="center" py="xl">
          <Loader size="lg" />
        </Stack>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <Group>
            <ActionIcon variant="subtle" onClick={() => navigate('/admin/catalogs')}>
              <IconArrowLeft size={20} />
            </ActionIcon>
            <Title order={2}>
              {isNew ? 'Neuer Kriterienkatalog' : `Katalog bearbeiten: ${name}`}
            </Title>
          </Group>
          <Stack gap="xs" style={{ minWidth: 300 }}>
            {catalog && (
              <Badge color={catalog.phase === 'draft' ? 'gray' : catalog.phase === 'active' ? 'blue' : 'orange'} size="lg">
                {catalog.phase === 'draft' ? 'Entwurf' : catalog.phase === 'active' ? 'Aktiv' : 'Archiviert'}
              </Badge>
            )}
            {!isNew && levels.length > 0 && categories.length > 0 && (
              <Stack gap="xs">
                {calculateCompletionStats().allCategoriesHavePaths ? (
                  <>
                    <Text size="sm" fw={500}>
                      Level-Beschreibungen Fortschritt
                    </Text>
                    <Progress
                      value={calculateCompletionStats().percentage}
                      color={calculateCompletionStats().percentage === 100 ? 'green' : 'blue'}
                      size="lg"
                    />
                    <Text size="xs" c="dimmed">
                      {calculateCompletionStats().filledDescriptions} von{' '}
                      {calculateCompletionStats().totalDescriptionsNeeded} Beschreibungen ausgefüllt (
                      {calculateCompletionStats().percentage}%)
                    </Text>
                  </>
                ) : (
                  <Alert color="yellow" p="xs">
                    <Text size="xs">
                      Es gibt Kategorien ohne Pfade
                    </Text>
                  </Alert>
                )}
              </Stack>
            )}
          </Stack>
        </Group>

        {error && (
          <Alert icon={<IconAlertCircle size={16} />} title="Fehler" color="red">
            {error}
          </Alert>
        )}

        <Tabs value={activeTab} onChange={setActiveTab}>
          <Tabs.List>
            <Tabs.Tab value="basic">Grunddaten</Tabs.Tab>
            {!isNew && <Tabs.Tab value="levels">Levels</Tabs.Tab>}
            {!isNew && <Tabs.Tab value="categories">Kategorien</Tabs.Tab>}
            {!isNew && <Tabs.Tab value="paths">Pfade</Tabs.Tab>}
          </Tabs.List>

          <Tabs.Panel value="basic" pt="md">
            <Paper p="md" withBorder>
              <Stack gap="md">
                <TextInput
                  label="Name"
                  placeholder="z.B. Kriterienkatalog 2025"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />

                <Textarea
                  label="Beschreibung"
                  placeholder="Optionale Beschreibung des Katalogs"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                />

                <Group gap="md">
                  <DatePickerInput
                    label="Gültig von"
                    placeholder="Startdatum"
                    value={validFrom}
                    onChange={(value) => setValidFrom(value as Date | null)}
                    required
                    valueFormat="DD.MM.YYYY"
                    clearable
                    style={{ flex: 1 }}
                  />
                  <DatePickerInput
                    label="Gültig bis"
                    placeholder="Enddatum"
                    value={validUntil}
                    onChange={(value) => setValidUntil(value as Date | null)}
                    required
                    valueFormat="DD.MM.YYYY"
                    clearable
                    style={{ flex: 1 }}
                  />
                </Group>

                {!isNew && (
                  <Select
                    label="Phase"
                    description={
                      !calculateCompletionStats().allCategoriesHavePaths
                        ? 'Jede Kategorie muss mindestens einen Pfad haben, um die Phase zu ändern'
                        : undefined
                    }
                    value={phase}
                    onChange={(value) => setPhase(value || 'draft')}
                    data={[
                      { value: 'draft', label: 'Entwurf' },
                      { value: 'active', label: 'Aktiv' },
                      { value: 'archived', label: 'Archiviert' },
                    ]}
                    disabled={!calculateCompletionStats().allCategoriesHavePaths}
                  />
                )}

                <Group justify="flex-end">
                  <Button
                    leftSection={<IconDeviceFloppy size={16} />}
                    onClick={handleSaveBasic}
                    loading={saving}
                  >
                    Speichern
                  </Button>
                </Group>
              </Stack>
            </Paper>
          </Tabs.Panel>

          <Tabs.Panel value="levels" pt="md">
            <Paper p="md" withBorder>
              <Stack gap="md">
                <Group justify="space-between">
                  <Title order={4}>Levels</Title>
                  <Button 
                    leftSection={<IconPlus size={16} />} 
                    onClick={handleOpenLevelModal}
                    disabled={phase !== 'draft'}
                  >
                    Level hinzufügen
                  </Button>
                </Group>

                {levels.length === 0 ? (
                  <Alert color="blue">Noch keine Levels vorhanden</Alert>
                ) : (
                  <Table>
                    <Table.Thead>
                      <Table.Tr>
                        <Table.Th>Nummer</Table.Th>
                        <Table.Th>Name</Table.Th>
                        <Table.Th>Beschreibung</Table.Th>
                        <Table.Th>Aktionen</Table.Th>
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {levels.map((level, index) => (
                        <Table.Tr key={level.id}>
                          <Table.Td>{level.level_number}</Table.Td>
                          <Table.Td>{level.name}</Table.Td>
                          <Table.Td>{level.description || '-'}</Table.Td>
                          <Table.Td>
                            <Group gap="xs">
                              <ActionIcon
                                variant="light"
                                color="gray"
                                onClick={() => handleMoveLevel(index, 'up')}
                                disabled={index === 0 || phase !== 'draft'}
                              >
                                <IconArrowUp size={16} />
                              </ActionIcon>
                              <ActionIcon
                                variant="light"
                                color="gray"
                                onClick={() => handleMoveLevel(index, 'down')}
                                disabled={index === levels.length - 1 || phase !== 'draft'}
                              >
                                <IconArrowDown size={16} />
                              </ActionIcon>
                              <ActionIcon
                                variant="light"
                                color="blue"
                                onClick={() => handleOpenEditLevelModal(level)}
                                disabled={phase !== 'draft'}
                              >
                                <IconEdit size={16} />
                              </ActionIcon>
                              <ActionIcon
                                variant="light"
                                color="red"
                                onClick={() => handleDeleteLevel(level.id)}
                                disabled={phase !== 'draft'}
                              >
                                <IconTrash size={16} />
                              </ActionIcon>
                            </Group>
                          </Table.Td>
                        </Table.Tr>
                      ))}
                    </Table.Tbody>
                  </Table>
                )}
              </Stack>
            </Paper>
          </Tabs.Panel>

          <Tabs.Panel value="categories" pt="md">
            <Paper p="md" withBorder>
              <Stack gap="md">
                <Group justify="space-between">
                  <Title order={4}>Kategorien</Title>
                  <Button 
                    leftSection={<IconPlus size={16} />} 
                    onClick={handleOpenCategoryModal}
                    disabled={phase !== 'draft'}
                  >
                    Kategorie hinzufügen
                  </Button>
                </Group>

                {categories.length === 0 ? (
                  <Alert color="blue">Noch keine Kategorien vorhanden</Alert>
                ) : (
                  <Table>
                    <Table.Thead>
                      <Table.Tr>
                        <Table.Th>Name</Table.Th>
                        <Table.Th>Beschreibung</Table.Th>
                        <Table.Th>Sortierung</Table.Th>
                        <Table.Th>Aktionen</Table.Th>
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {categories.map((category, index) => (
                        <Table.Tr key={category.id}>
                          <Table.Td>{category.name}</Table.Td>
                          <Table.Td>{category.description || '-'}</Table.Td>
                          <Table.Td>{category.sort_order}</Table.Td>
                          <Table.Td>
                            <Group gap="xs">
                              <ActionIcon
                                variant="light"
                                color="gray"
                                onClick={() => handleMoveCategory(index, 'up')}
                                disabled={index === 0 || phase !== 'draft'}
                              >
                                <IconArrowUp size={16} />
                              </ActionIcon>
                              <ActionIcon
                                variant="light"
                                color="gray"
                                onClick={() => handleMoveCategory(index, 'down')}
                                disabled={index === categories.length - 1 || phase !== 'draft'}
                              >
                                <IconArrowDown size={16} />
                              </ActionIcon>
                              <ActionIcon
                                variant="light"
                                color="blue"
                                onClick={() => handleOpenEditCategoryModal(category)}
                                disabled={phase !== 'draft'}
                              >
                                <IconEdit size={16} />
                              </ActionIcon>
                              <ActionIcon
                                variant="light"
                                color="red"
                                onClick={() => handleDeleteCategory(category.id)}
                                disabled={phase !== 'draft'}
                              >
                                <IconTrash size={16} />
                              </ActionIcon>
                            </Group>
                          </Table.Td>
                        </Table.Tr>
                      ))}
                    </Table.Tbody>
                  </Table>
                )}
              </Stack>
            </Paper>
          </Tabs.Panel>

          <Tabs.Panel value="paths" pt="md">
            <PathManagement
              catalogId={parseInt(id!, 10)}
              categories={categories}
              levels={levels}
              phase={phase}
              onUpdate={() => loadCatalog(parseInt(id!, 10))}
            />
          </Tabs.Panel>
        </Tabs>

        {/* Level Modal */}
        <Modal
          opened={levelModalOpened}
          onClose={() => {
            setLevelModalOpened(false);
            setEditingLevel(null);
            setNewLevelName('');
            setNewLevelDescription('');
          }}
          title={editingLevel ? 'Level bearbeiten' : 'Neues Level hinzufügen'}
        >
          <form onSubmit={(e) => {
            e.preventDefault();
            if (newLevelName.trim()) {
              handleSaveLevel();
            }
          }}>
          <Stack gap="md">
            <TextInput
              label="Level-Name"
              placeholder="z.B. Junior, Senior, Expert"
              value={newLevelName}
              onChange={(e) => setNewLevelName(e.target.value)}
              required
              data-autofocus
            />
            <Textarea
              label="Beschreibung"
              placeholder="Optionale Beschreibung des Levels"
              value={newLevelDescription}
              onChange={(e) => setNewLevelDescription(e.target.value)}
              rows={3}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault();
                  if (newLevelName.trim()) {
                    handleSaveLevel();
                  }
                }
              }}
            />
            <Group justify="flex-end">
              <Button
                variant="subtle"
                onClick={() => {
                  setLevelModalOpened(false);
                  setEditingLevel(null);
                  setNewLevelName('');
                  setNewLevelDescription('');
                }}
                type="button"
              >
                Abbrechen
              </Button>
              <Button
                type="submit"
                disabled={!newLevelName.trim()}
              >
                {editingLevel ? 'Speichern' : 'Hinzufügen'}
              </Button>
            </Group>
          </Stack>
          </form>
        </Modal>

        {/* Category Modal */}
        <Modal
          opened={categoryModalOpened}
          onClose={() => {
            setCategoryModalOpened(false);
            setEditingCategory(null);
            setNewCategoryName('');
            setNewCategoryDescription('');
          }}
          title={editingCategory ? 'Kategorie bearbeiten' : 'Neue Kategorie hinzufügen'}
        >
          <form onSubmit={(e) => {
            e.preventDefault();
            if (newCategoryName.trim()) {
              handleSaveCategory();
            }
          }}>
          <Stack gap="md">
            <TextInput
              label="Kategorie-Name"
              placeholder="z.B. Fachkompetenz, Sozialkompetenz"
              value={newCategoryName}
              onChange={(e) => setNewCategoryName(e.target.value)}
              required
              data-autofocus
            />
            <Textarea
              label="Beschreibung"
              placeholder="Optionale Beschreibung der Kategorie"
              value={newCategoryDescription}
              onChange={(e) => setNewCategoryDescription(e.target.value)}
              rows={3}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault();
                  if (newCategoryName.trim()) {
                    handleSaveCategory();
                  }
                }
              }}
            />
            <Group justify="flex-end">
              <Button
                variant="subtle"
                onClick={() => {
                  setCategoryModalOpened(false);
                  setEditingCategory(null);
                  setNewCategoryName('');
                  setNewCategoryDescription('');
                }}
                type="button"
              >
                Abbrechen
              </Button>
              <Button
                type="submit"
                disabled={!newCategoryName.trim()}
              >
                {editingCategory ? 'Speichern' : 'Hinzufügen'}
              </Button>
            </Group>
          </Stack>
          </form>
        </Modal>
      </Stack>
    </Container>
  );
}
