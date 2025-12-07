import { useState, useEffect } from 'react';
import {
  Container,
  Paper,
  Title,
  Table,
  Badge,
  Group,
  Pagination,
  Text,
  Code,
  TextInput,
  Grid,
  Button,
  ActionIcon,
  Select,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconSearch, IconX, IconChevronUp, IconChevronDown } from '@tabler/icons-react';
import { adminApi, type AuditLogListParams } from '../../services/admin';
import type { AuditLog, ApiError } from '../../types';

export const AuditLogsPage = () => {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalCount, setTotalCount] = useState(0);
  const [pageSize, setPageSize] = useState(50);

  // Filter states
  const [actionFilter, setActionFilter] = useState('');
  const [resourceFilter, setResourceFilter] = useState('');
  const [sortBy, setSortBy] = useState<string>('created_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  useEffect(() => {
    loadLogs();
  }, [currentPage, actionFilter, resourceFilter, sortBy, sortOrder, pageSize]);

  const loadLogs = async () => {
    setIsLoading(true);
    try {
      const params: AuditLogListParams = {
        page: currentPage,
        limit: pageSize,
        sort_by: sortBy,
        sort_order: sortOrder,
      };

      if (actionFilter) params.action = actionFilter;
      if (resourceFilter) params.resource = resourceFilter;

      const data = await adminApi.listAuditLogs(params);
      setLogs(data.logs || []);
      setTotalPages(data.total_pages);
      setTotalCount(data.total);
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to load audit logs',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const getActionColor = (action: string): string => {
    if (action.includes('login')) return 'blue';
    if (action.includes('register')) return 'green';
    if (action.includes('delete') || action.includes('remove')) return 'red';
    if (action.includes('update') || action.includes('assign')) return 'yellow';
    return 'gray';
  };

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(column);
      setSortOrder('desc');
    }
  };

  const clearFilters = () => {
    setActionFilter('');
    setResourceFilter('');
    setCurrentPage(1);
  };

  const handlePageSizeChange = (value: string | null) => {
    if (value) {
      setPageSize(parseInt(value));
      setCurrentPage(1);
    }
  };

  const SortIcon = ({ column }: { column: string }) => {
    if (sortBy !== column) return null;
    return sortOrder === 'asc' ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />;
  };

  return (
    <Container size="xl" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Title order={2} mb="xl">Audit Logs</Title>

        {/* Filters */}
        <Paper withBorder p="md" mb="md" bg="dark.6">
          <Grid>
            <Grid.Col span={{ base: 12, md: 6 }}>
              <TextInput
                placeholder="Filter by action"
                leftSection={<IconSearch size={16} />}
                value={actionFilter}
                onChange={(e) => {
                  setActionFilter(e.target.value);
                  setCurrentPage(1);
                }}
                rightSection={
                  actionFilter ? (
                    <ActionIcon size="sm" variant="subtle" onClick={() => setActionFilter('')}>
                      <IconX size={14} />
                    </ActionIcon>
                  ) : null
                }
              />
            </Grid.Col>
            <Grid.Col span={{ base: 12, md: 6 }}>
              <TextInput
                placeholder="Filter by resource"
                leftSection={<IconSearch size={16} />}
                value={resourceFilter}
                onChange={(e) => {
                  setResourceFilter(e.target.value);
                  setCurrentPage(1);
                }}
                rightSection={
                  resourceFilter ? (
                    <ActionIcon size="sm" variant="subtle" onClick={() => setResourceFilter('')}>
                      <IconX size={14} />
                    </ActionIcon>
                  ) : null
                }
              />
            </Grid.Col>
          </Grid>

          <Group gap="md" mt="md" justify="space-between">
            <Text size="sm" c="dimmed">
              Showing {logs.length} of {totalCount} logs
            </Text>
            {(actionFilter || resourceFilter) && (
              <Button variant="subtle" onClick={clearFilters} leftSection={<IconX size={14} />}>
                Clear filters
              </Button>
            )}
          </Group>
        </Paper>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('created_at')}>
                <Group gap="xs">
                  Timestamp <SortIcon column="created_at" />
                </Group>
              </Table.Th>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('user_id')}>
                <Group gap="xs">
                  User <SortIcon column="user_id" />
                </Group>
              </Table.Th>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('action')}>
                <Group gap="xs">
                  Action <SortIcon column="action" />
                </Group>
              </Table.Th>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('resource')}>
                <Group gap="xs">
                  Resource <SortIcon column="resource" />
                </Group>
              </Table.Th>
              <Table.Th>Details</Table.Th>
              <Table.Th>IP Address</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {isLoading ? (
              <Table.Tr>
                <Table.Td colSpan={6} style={{ textAlign: 'center' }}>
                  Loading...
                </Table.Td>
              </Table.Tr>
            ) : logs.length === 0 ? (
              <Table.Tr>
                <Table.Td colSpan={6} style={{ textAlign: 'center' }}>
                  No audit logs found
                </Table.Td>
              </Table.Tr>
            ) : (
              logs.map((log) => (
                <Table.Tr key={log.id}>
                  <Table.Td>
                    <Text size="sm">
                      {new Date(log.created_at).toLocaleString('de-DE', {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit',
                      })}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    {log.user_id ? (
                      <div>
                        <Text size="sm" fw={500}>
                          {log.user_email || 'Unknown'}
                        </Text>
                        <Text size="xs" c="dimmed">
                          ID: {log.user_id}
                        </Text>
                      </div>
                    ) : (
                      <Text c="dimmed" size="sm">N/A</Text>
                    )}
                  </Table.Td>
                  <Table.Td>
                    <Badge color={getActionColor(log.action)} variant="light">
                      {log.action}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Code>{log.resource}</Code>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" lineClamp={2}>
                      {log.details}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">
                      {log.ip_address}
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ))
            )}
          </Table.Tbody>
        </Table>

        <Group justify="space-between" mt="xl" align="center">
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              Zeige
            </Text>
            <Select
              value={pageSize.toString()}
              onChange={handlePageSizeChange}
              data={[
                { value: '25', label: '25' },
                { value: '50', label: '50' },
                { value: '100', label: '100' },
                { value: '250', label: '250' },
              ]}
              w={70}
              size="sm"
            />
            <Text size="sm" c="dimmed">
              von {totalCount} Einträgen | Seite {currentPage} von {totalPages}
            </Text>
          </Group>
          <Pagination
            value={currentPage}
            onChange={setCurrentPage}
            total={totalPages}
          />
        </Group>
      </Paper>
    </Container>
  );
};
