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
} from '@mantine/core';
import {
  IconAlertCircle,
  IconDeviceFloppy,
  IconArrowLeft,
  IconPlus,
  IconTrash,
} from '@tabler/icons-react';
import { DateInput } from '@mantine/dates';
import { notifications } from '@mantine/notifications';
import { adminApi } from '../../services/admin';
import type {
  CatalogWithDetails,
  Category,
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

  // Nested entities
  const [catalog, setCatalog] = useState<CatalogWithDetails | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [levels, setLevels] = useState<Level[]>([]);

  // Active tab
  const [activeTab, setActiveTab] = useState<string | null>('basic');

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
      setCategories(data.categories?.map((c) => c) || []);
      setLevels(data.levels || []);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load catalog');
    } finally {
      setLoading(false);
    }
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
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || err.message || 'Fehler beim Speichern',
        color: 'red',
      });
    } finally {
      setSaving(false);
    }
  };

  const handleAddLevel = async () => {
    if (!catalog) return;

    const levelNumber = levels.length + 1;
    const levelName = `Level ${levelNumber}`;

    try {
      const newLevel = await adminApi.createLevel(catalog.id, {
        name: levelName,
        level_number: levelNumber,
        description: '',
      });
      setLevels([...levels, newLevel]);
      notifications.show({
        title: 'Erfolg',
        message: 'Level erfolgreich hinzugefügt',
        color: 'green',
      });
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Hinzufügen',
        color: 'red',
      });
    }
  };

  const handleDeleteLevel = async (levelId: number) => {
    if (!catalog) return;
    if (!confirm('Level wirklich löschen?')) return;

    try {
      await adminApi.deleteLevel(catalog.id, levelId);
      setLevels(levels.filter((l) => l.id !== levelId));
      notifications.show({
        title: 'Erfolg',
        message: 'Level erfolgreich gelöscht',
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

  const handleAddCategory = async () => {
    if (!catalog) return;

    const categoryName = `Kategorie ${categories.length + 1}`;

    try {
      const newCategory = await adminApi.createCategory(catalog.id, {
        name: categoryName,
        description: '',
        sort_order: categories.length,
      });
      setCategories([...categories, newCategory]);
      notifications.show({
        title: 'Erfolg',
        message: 'Kategorie erfolgreich hinzugefügt',
        color: 'green',
      });
    } catch (err: any) {
      notifications.show({
        title: 'Fehler',
        message: err.response?.data?.error || 'Fehler beim Hinzufügen',
        color: 'red',
      });
    }
  };

  const handleDeleteCategory = async (categoryId: number) => {
    if (!catalog) return;
    if (!confirm('Kategorie wirklich löschen?')) return;

    try {
      await adminApi.deleteCategory(catalog.id, categoryId);
      setCategories(categories.filter((c) => c.id !== categoryId));
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
          {catalog && (
            <Badge color={catalog.phase === 'draft' ? 'gray' : catalog.phase === 'review' ? 'blue' : 'orange'}>
              {catalog.phase === 'draft' ? 'Entwurf' : catalog.phase === 'review' ? 'Haupt' : 'Abschluss'}
            </Badge>
          )}
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

                <Group grow>
                  <DateInput
                    label="Gültig von"
                    placeholder="Startdatum"
                    value={validFrom}
                    onChange={(value) => setValidFrom(value as Date | null)}
                    required
                    valueFormat="DD.MM.YYYY"
                    clearable
                  />
                  <DateInput
                    label="Gültig bis"
                    placeholder="Enddatum"
                    value={validUntil}
                    onChange={(value) => setValidUntil(value as Date | null)}
                    required
                    valueFormat="DD.MM.YYYY"
                    clearable
                  />
                </Group>

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
                  <Button leftSection={<IconPlus size={16} />} onClick={handleAddLevel}>
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
                      {levels.map((level) => (
                        <Table.Tr key={level.id}>
                          <Table.Td>{level.level_number}</Table.Td>
                          <Table.Td>{level.name}</Table.Td>
                          <Table.Td>{level.description || '-'}</Table.Td>
                          <Table.Td>
                            <Group gap="xs">
                              <ActionIcon
                                variant="light"
                                color="red"
                                onClick={() => handleDeleteLevel(level.id)}
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
                  <Button leftSection={<IconPlus size={16} />} onClick={handleAddCategory}>
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
                      {categories.map((category) => (
                        <Table.Tr key={category.id}>
                          <Table.Td>{category.name}</Table.Td>
                          <Table.Td>{category.description || '-'}</Table.Td>
                          <Table.Td>{category.sort_order}</Table.Td>
                          <Table.Td>
                            <Group gap="xs">
                              <ActionIcon
                                variant="light"
                                color="red"
                                onClick={() => handleDeleteCategory(category.id)}
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
        </Tabs>
      </Stack>
    </Container>
  );
}
