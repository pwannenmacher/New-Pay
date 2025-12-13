import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Table,
  Badge,
  Stack,
  Text,
  ActionIcon,
  Loader,
  Alert,
} from '@mantine/core';
import { IconEye, IconAlertCircle } from '@tabler/icons-react';
import { adminApi } from '../services/admin';
import type { CriteriaCatalog, CatalogPhase } from '../types';

const getPhaseColor = (phase: CatalogPhase): string => {
  switch (phase) {
    case 'draft':
      return 'gray';
    case 'active':
      return 'blue';
    case 'archived':
      return 'green';
    default:
      return 'gray';
  }
};

const getPhaseLabel = (phase: CatalogPhase): string => {
  switch (phase) {
    case 'draft':
      return 'Entwurf';
    case 'active':
      return 'Aktiv';
    case 'archived':
      return 'Archiviert';
    default:
      return phase;
  }
};

export function CatalogsPage() {
  const navigate = useNavigate();
  const [catalogs, setCatalogs] = useState<CriteriaCatalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadCatalogs();
  }, []);

  const loadCatalogs = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await adminApi.listCatalogs();
      setCatalogs(data);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Fehler beim Laden der Kriterienkataloge');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('de-DE');
  };

  if (loading) {
    return (
      <Container size="xl" py="xl">
        <Stack align="center" py="xl">
          <Loader size="lg" />
          <Text>Kriterienkataloge werden geladen...</Text>
        </Stack>
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Title order={2}>Kriterienkataloge</Title>

        {error && (
          <Alert icon={<IconAlertCircle size={16} />} title="Fehler" color="red">
            {error}
          </Alert>
        )}

        {catalogs.length === 0 ? (
          <Alert icon={<IconAlertCircle size={16} />} title="Keine Kataloge" color="blue">
            Aktuell sind keine Kriterienkataloge verfügbar.
          </Alert>
        ) : (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Gültig von</Table.Th>
                <Table.Th>Gültig bis</Table.Th>
                <Table.Th>Phase</Table.Th>
                <Table.Th>Aktion</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {catalogs.map((catalog) => (
                <Table.Tr key={catalog.id}>
                  <Table.Td>
                    <Text fw={500}>{catalog.name}</Text>
                    {catalog.description && (
                      <Text size="sm" c="dimmed">
                        {catalog.description}
                      </Text>
                    )}
                  </Table.Td>
                  <Table.Td>{formatDate(catalog.valid_from)}</Table.Td>
                  <Table.Td>{formatDate(catalog.valid_until)}</Table.Td>
                  <Table.Td>
                    <Badge color={getPhaseColor(catalog.phase)}>
                      {getPhaseLabel(catalog.phase)}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <ActionIcon
                      variant="light"
                      color="blue"
                      onClick={() => navigate(`/admin/catalogs/${catalog.id}`)}
                    >
                      <IconEye size={16} />
                    </ActionIcon>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Stack>
    </Container>
  );
}
