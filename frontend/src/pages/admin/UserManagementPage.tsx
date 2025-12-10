import { useState, useEffect } from 'react';
import {
  Container,
  Paper,
  Title,
  Table,
  Badge,
  Button,
  Group,
  Pagination,
  Select,
  Modal,
  Stack,
  Text,
  ActionIcon,
  Tooltip,
  TextInput,
  MultiSelect,
  Switch,
  Grid,
  PasswordInput,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconUserPlus, IconUserMinus, IconSearch, IconX, IconChevronUp, IconChevronDown, IconLock, IconLockOpen, IconEdit, IconTrash, IconKey, IconMail, IconMailOff, IconMailX } from '@tabler/icons-react';
import { adminApi, type UserListParams } from '../../services/admin';
import type { UserWithRoles, Role, ApiError } from '../../types';

export const UserManagementPage = () => {
  const [users, setUsers] = useState<UserWithRoles[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalCount, setTotalCount] = useState(0);
  const [pageSize, setPageSize] = useState(20);
  const [modalOpened, setModalOpened] = useState(false);
  const [selectedUser, setSelectedUser] = useState<UserWithRoles | null>(null);
  const [selectedRoleId, setSelectedRoleId] = useState<string | null>(null);
  const [actionType, setActionType] = useState<'assign' | 'remove'>('assign');
  const [actionLoading, setActionLoading] = useState(false);

  // Edit user modal states
  const [editModalOpened, setEditModalOpened] = useState(false);
  const [editEmail, setEditEmail] = useState('');
  const [editFirstName, setEditFirstName] = useState('');
  const [editLastName, setEditLastName] = useState('');

  // Password modal states
  const [passwordModalOpened, setPasswordModalOpened] = useState(false);
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  // Delete modal states
  const [deleteModalOpened, setDeleteModalOpened] = useState(false);

  // Filter states
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRoleFilters, setSelectedRoleFilters] = useState<string[]>([]);
  const [activeFilter, setActiveFilter] = useState<boolean | undefined>(undefined);
  const [verifiedFilter, setVerifiedFilter] = useState<boolean | undefined>(undefined);
  const [sortBy, setSortBy] = useState<string>('created_at');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  useEffect(() => {
    loadUsers();
  }, [currentPage, searchQuery, selectedRoleFilters, activeFilter, verifiedFilter, sortBy, sortOrder, pageSize]);

  useEffect(() => {
    loadRoles();
  }, []);

  const loadUsers = async () => {
    setIsLoading(true);
    try {
      const params: UserListParams = {
        page: currentPage,
        limit: pageSize,
        sort_by: sortBy,
        sort_order: sortOrder,
      };

      if (searchQuery) params.search = searchQuery;
      if (selectedRoleFilters.length > 0) {
        params.role_ids = selectedRoleFilters.map(id => parseInt(id));
      }
      if (activeFilter !== undefined) params.is_active = activeFilter;
      if (verifiedFilter !== undefined) params.email_verified = verifiedFilter;

      const data = await adminApi.listUsers(params);
      setUsers(data.users || []);
      setTotalPages(data.total_pages);
      setTotalCount(data.total);
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to load users',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const loadRoles = async () => {
    try {
      const data = await adminApi.listRoles();
      setRoles(data);
    } catch (error) {
      console.error('Failed to load roles:', error);
    }
  };

  const openAssignRoleModal = (user: UserWithRoles) => {
    setSelectedUser(user);
    setActionType('assign');
    setSelectedRoleId(null);
    setModalOpened(true);
  };

  const openRemoveRoleModal = (user: UserWithRoles) => {
    setSelectedUser(user);
    setActionType('remove');
    setSelectedRoleId(null);
    setModalOpened(true);
  };

  const handleToggleActiveStatus = async (user: UserWithRoles) => {
    try {
      await adminApi.updateUserStatus(user.id, !user.is_active);
      notifications.show({
        title: 'Success',
        message: `User ${user.is_active ? 'deactivated' : 'activated'} successfully`,
        color: 'green',
      });
      loadUsers();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to update user status',
        color: 'red',
      });
    }
  };

  const handleRoleAction = async () => {
    if (!selectedUser || !selectedRoleId) {
      notifications.show({
        title: 'Error',
        message: 'Please select a role',
        color: 'red',
      });
      return;
    }

    setActionLoading(true);
    try {
      if (actionType === 'assign') {
        await adminApi.assignRole({
          user_id: selectedUser.id,
          role_id: parseInt(selectedRoleId),
        });
        notifications.show({
          title: 'Success',
          message: 'Role assigned successfully',
          color: 'green',
        });
      } else {
        await adminApi.removeRole({
          user_id: selectedUser.id,
          role_id: parseInt(selectedRoleId),
        });
        notifications.show({
          title: 'Success',
          message: 'Role removed successfully',
          color: 'green',
        });
      }
      setModalOpened(false);
      loadUsers();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || `Failed to ${actionType} role`,
        color: 'red',
      });
    } finally {
      setActionLoading(false);
    }
  };

  const handleSort = (column: string) => {
    if (sortBy === column) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(column);
      setSortOrder('desc');
    }
  };

  const openEditModal = (user: UserWithRoles) => {
    setSelectedUser(user);
    setEditEmail(user.email);
    setEditFirstName(user.first_name);
    setEditLastName(user.last_name);
    setEditModalOpened(true);
  };

  const handleUpdateUser = async () => {
    if (!selectedUser) return;

    setActionLoading(true);
    try {
      await adminApi.updateUser(selectedUser.id, editEmail, editFirstName, editLastName);
      notifications.show({
        title: 'Success',
        message: 'User updated successfully',
        color: 'green',
      });
      setEditModalOpened(false);
      loadUsers();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to update user',
        color: 'red',
      });
    } finally {
      setActionLoading(false);
    }
  };

  const openPasswordModal = (user: UserWithRoles) => {
    setSelectedUser(user);
    setNewPassword('');
    setConfirmPassword('');
    setPasswordModalOpened(true);
  };

  const handleSetPassword = async () => {
    if (!selectedUser) return;

    if (newPassword.length < 8) {
      notifications.show({
        title: 'Error',
        message: 'Password must be at least 8 characters long',
        color: 'red',
      });
      return;
    }

    if (newPassword !== confirmPassword) {
      notifications.show({
        title: 'Error',
        message: 'Passwords do not match',
        color: 'red',
      });
      return;
    }

    setActionLoading(true);
    try {
      await adminApi.setUserPassword(selectedUser.id, newPassword);
      notifications.show({
        title: 'Success',
        message: 'Password updated successfully',
        color: 'green',
      });
      setPasswordModalOpened(false);
      setNewPassword('');
      setConfirmPassword('');
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to set password',
        color: 'red',
      });
    } finally {
      setActionLoading(false);
    }
  };

  const openDeleteModal = (user: UserWithRoles) => {
    setSelectedUser(user);
    setDeleteModalOpened(true);
  };

  const handleDeleteUser = async () => {
    if (!selectedUser) return;

    setActionLoading(true);
    try {
      await adminApi.deleteUser(selectedUser.id);
      notifications.show({
        title: 'Success',
        message: 'User deleted successfully',
        color: 'green',
      });
      setDeleteModalOpened(false);
      loadUsers();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to delete user',
        color: 'red',
      });
    } finally {
      setActionLoading(false);
    }
  };

  const handleSendVerification = async (user: UserWithRoles) => {
    try {
      await adminApi.sendVerificationEmail(user.id);
      notifications.show({
        title: 'Success',
        message: 'Verification email sent successfully',
        color: 'green',
      });
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to send verification email',
        color: 'red',
      });
    }
  };

  const handleCancelVerification = async (user: UserWithRoles) => {
    try {
      await adminApi.cancelVerification(user.id);
      notifications.show({
        title: 'Success',
        message: 'Pending verification cancelled successfully',
        color: 'green',
      });
      loadUsers();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to cancel verification',
        color: 'red',
      });
    }
  };

  const handleRevokeVerification = async (user: UserWithRoles) => {
    try {
      await adminApi.revokeVerification(user.id);
      notifications.show({
        title: 'Success',
        message: 'Email verification revoked successfully',
        color: 'green',
      });
      loadUsers();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to revoke verification',
        color: 'red',
      });
    }
  };

  const clearFilters = () => {
    setSearchQuery('');
    setSelectedRoleFilters([]);
    setActiveFilter(undefined);
    setVerifiedFilter(undefined);
    setCurrentPage(1);
  };

  const handlePageSizeChange = (value: string | null) => {
    if (value) {
      setPageSize(parseInt(value));
      setCurrentPage(1);
    }
  };

  const availableRoles = actionType === 'assign'
    ? roles.filter(role => !selectedUser?.roles?.some(ur => ur.id === role.id))
    : selectedUser?.roles || [];

  const SortIcon = ({ column }: { column: string }) => {
    if (sortBy !== column) return null;
    return sortOrder === 'asc' ? <IconChevronUp size={14} /> : <IconChevronDown size={14} />;
  };

  return (
    <Container size="xl" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Title order={2} mb="xl">Benutzerverwaltung</Title>

        {/* Filters */}
        <Paper withBorder p="md" mb="md" bg="dark.6">
          <Stack gap="md">
            <Grid>
              <Grid.Col span={{ base: 12, md: 6 }}>
                <TextInput
                  placeholder="Search by name or email"
                  leftSection={<IconSearch size={16} />}
                  value={searchQuery}
                  onChange={(e) => {
                    setSearchQuery(e.target.value);
                    setCurrentPage(1);
                  }}
                  rightSection={
                    searchQuery ? (
                      <ActionIcon size="sm" variant="subtle" onClick={() => setSearchQuery('')}>
                        <IconX size={14} />
                      </ActionIcon>
                    ) : null
                  }
                />
              </Grid.Col>
              <Grid.Col span={{ base: 12, md: 6 }}>
                <MultiSelect
                  placeholder="Filter by roles"
                  data={roles.map(role => ({ value: role.id.toString(), label: role.name }))}
                  value={selectedRoleFilters}
                  onChange={(values) => {
                    setSelectedRoleFilters(values);
                    setCurrentPage(1);
                  }}
                  clearable
                />
              </Grid.Col>
            </Grid>

            <Group gap="md">
              <Switch
                label="Active only"
                checked={activeFilter === true}
                onChange={(e) => {
                  setActiveFilter(e.currentTarget.checked ? true : undefined);
                  setCurrentPage(1);
                }}
              />
              <Switch
                label="Inactive only"
                checked={activeFilter === false}
                onChange={(e) => {
                  setActiveFilter(e.currentTarget.checked ? false : undefined);
                  setCurrentPage(1);
                }}
              />
              <Switch
                label="Verified only"
                checked={verifiedFilter === true}
                onChange={(e) => {
                  setVerifiedFilter(e.currentTarget.checked ? true : undefined);
                  setCurrentPage(1);
                }}
              />
              <Switch
                label="Unverified only"
                checked={verifiedFilter === false}
                onChange={(e) => {
                  setVerifiedFilter(e.currentTarget.checked ? false : undefined);
                  setCurrentPage(1);
                }}
              />
              {(searchQuery || selectedRoleFilters.length > 0 || activeFilter !== undefined || verifiedFilter !== undefined) && (
                <Button variant="subtle" onClick={clearFilters} leftSection={<IconX size={14} />}>
                  Clear filters
                </Button>
              )}
            </Group>

            <Text size="sm" c="dimmed">
              Showing {users.length} of {totalCount} users
            </Text>
          </Stack>
        </Paper>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('id')}>
                <Group gap="xs">
                  ID <SortIcon column="id" />
                </Group>
              </Table.Th>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('name')}>
                <Group gap="xs">
                  Name <SortIcon column="name" />
                </Group>
              </Table.Th>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('email')}>
                <Group gap="xs">
                  Email <SortIcon column="email" />
                </Group>
              </Table.Th>
              <Table.Th>Login Methods</Table.Th>
              <Table.Th>Roles</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th style={{ cursor: 'pointer' }} onClick={() => handleSort('last_login_at')}>
                <Group gap="xs">
                  Last Login <SortIcon column="last_login_at" />
                </Group>
              </Table.Th>
              <Table.Th>Actions</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {isLoading ? (
              <Table.Tr>
                <Table.Td colSpan={8} style={{ textAlign: 'center' }}>
                  Loading...
                </Table.Td>
              </Table.Tr>
            ) : users.length === 0 ? (
              <Table.Tr>
                <Table.Td colSpan={8} style={{ textAlign: 'center' }}>
                  No users found
                </Table.Td>
              </Table.Tr>
            ) : (
              users.map((user) => (
                <Table.Tr key={user.id}>
                  <Table.Td>{user.id}</Table.Td>
                  <Table.Td>
                    <div>{user.first_name} {user.last_name}</div>
                  </Table.Td>
                  <Table.Td>
                    <Stack gap={4}>
                      <Text size="sm">{user.email}</Text>
                      {user.email_verified ? (
                        <Badge size="xs" color="green" variant="dot">Verified</Badge>
                      ) : (
                        <Badge size="xs" color="gray" variant="dot">Not verified</Badge>
                      )}
                    </Stack>
                  </Table.Td>
                  <Table.Td>
                    <Stack gap={4}>
                      {user.has_local_password && (
                        <Badge size="xs" variant="light" color="blue">Email/Password</Badge>
                      )}
                      {user.oauth_connections && user.oauth_connections.length > 0 ? (
                        user.oauth_connections.map((conn) => (
                          <Badge key={conn.id} size="xs" variant="light" color="grape">
                            {conn.provider}
                          </Badge>
                        ))
                      ) : null}
                      {!user.has_local_password && (!user.oauth_connections || user.oauth_connections.length === 0) && (
                        <Text c="dimmed" size="xs">No login method</Text>
                      )}
                    </Stack>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      {user.roles && user.roles.length > 0 ? (
                        user.roles.map((role) => (
                          <Badge key={role.id} variant="light">
                            {role.name}
                          </Badge>
                        ))
                      ) : (
                        <Text c="dimmed" size="sm">No roles</Text>
                      )}
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Badge color={user.is_active ? 'green' : 'red'} variant="light">
                      {user.is_active ? 'Active' : 'Inactive'}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">
                      {user.last_login_at 
                        ? new Date(user.last_login_at).toLocaleString('de-DE', {
                            dateStyle: 'short',
                            timeStyle: 'short'
                          })
                        : 'Never'}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <Tooltip label="Edit User">
                        <ActionIcon
                          color="blue"
                          variant="light"
                          onClick={() => openEditModal(user)}
                        >
                          <IconEdit size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Set Password">
                        <ActionIcon
                          color="violet"
                          variant="light"
                          onClick={() => openPasswordModal(user)}
                        >
                          <IconKey size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label={user.is_active ? 'Deactivate User' : 'Activate User'}>
                        <ActionIcon
                          color={user.is_active ? 'orange' : 'green'}
                          variant="light"
                          onClick={() => handleToggleActiveStatus(user)}
                        >
                          {user.is_active ? <IconLock size={16} /> : <IconLockOpen size={16} />}
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Send Verification Email">
                        <ActionIcon
                          color="cyan"
                          variant="light"
                          onClick={() => handleSendVerification(user)}
                        >
                          <IconMail size={16} />
                        </ActionIcon>
                      </Tooltip>
                      {!user.email_verified && (
                        <Tooltip label="Cancel Pending Verification">
                          <ActionIcon
                            color="orange"
                            variant="light"
                            onClick={() => handleCancelVerification(user)}
                          >
                            <IconMailX size={16} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      {user.email_verified && (
                        <Tooltip label="Revoke Email Verification">
                          <ActionIcon
                            color="red"
                            variant="light"
                            onClick={() => handleRevokeVerification(user)}
                          >
                            <IconMailOff size={16} />
                          </ActionIcon>
                        </Tooltip>
                      )}
                      <Tooltip label="Assign Role">
                        <ActionIcon
                          color="blue"
                          variant="light"
                          onClick={() => openAssignRoleModal(user)}
                        >
                          <IconUserPlus size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Remove Role">
                        <ActionIcon
                          color="red"
                          variant="light"
                          onClick={() => openRemoveRoleModal(user)}
                          disabled={!user.roles || user.roles.length === 0}
                        >
                          <IconUserMinus size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Tooltip label="Delete User">
                        <ActionIcon
                          color="red"
                          variant="filled"
                          onClick={() => openDeleteModal(user)}
                        >
                          <IconTrash size={16} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
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
                { value: '10', label: '10' },
                { value: '20', label: '20' },
                { value: '50', label: '50' },
                { value: '100', label: '100' },
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

      {/* Role Action Modal */}
      <Modal
        opened={modalOpened}
        onClose={() => setModalOpened(false)}
        title={actionType === 'assign' ? 'Assign Role' : 'Remove Role'}
      >
        <Stack>
          {selectedUser && (
            <Text>
              {actionType === 'assign' ? 'Assign role to' : 'Remove role from'}: {selectedUser.first_name} {selectedUser.last_name}
            </Text>
          )}

          <Select
            label="Select Role"
            placeholder="Choose a role"
            data={availableRoles.map((role) => ({
              value: role.id.toString(),
              label: role.name,
            }))}
            value={selectedRoleId}
            onChange={setSelectedRoleId}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={() => setModalOpened(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleRoleAction}
              loading={actionLoading}
              color={actionType === 'assign' ? 'blue' : 'red'}
            >
              {actionType === 'assign' ? 'Assign' : 'Remove'}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Edit User Modal */}
      <Modal
        opened={editModalOpened}
        onClose={() => setEditModalOpened(false)}
        title="Edit User"
      >
        <Stack>
          {selectedUser && (
            <Text size="sm" c="dimmed">
              Editing: {selectedUser.first_name} {selectedUser.last_name}
            </Text>
          )}

          <TextInput
            label="Email"
            placeholder="user@example.com"
            value={editEmail}
            onChange={(e) => setEditEmail(e.currentTarget.value)}
            required
          />

          <TextInput
            label="First Name"
            placeholder="John"
            value={editFirstName}
            onChange={(e) => setEditFirstName(e.currentTarget.value)}
            required
          />

          <TextInput
            label="Last Name"
            placeholder="Doe"
            value={editLastName}
            onChange={(e) => setEditLastName(e.currentTarget.value)}
            required
          />

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={() => setEditModalOpened(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleUpdateUser}
              loading={actionLoading}
            >
              Update
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Set Password Modal */}
      <Modal
        opened={passwordModalOpened}
        onClose={() => setPasswordModalOpened(false)}
        title="Set New Password"
      >
        <Stack>
          {selectedUser && (
            <Text size="sm" c="dimmed">
              Setting password for: {selectedUser.first_name} {selectedUser.last_name}
            </Text>
          )}

          <PasswordInput
            label="New Password"
            placeholder="Minimum 8 characters"
            value={newPassword}
            onChange={(e) => setNewPassword(e.currentTarget.value)}
            required
          />

          <PasswordInput
            label="Confirm Password"
            placeholder="Repeat password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.currentTarget.value)}
            required
          />

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={() => setPasswordModalOpened(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSetPassword}
              loading={actionLoading}
              color="violet"
            >
              Set Password
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete User Modal */}
      <Modal
        opened={deleteModalOpened}
        onClose={() => setDeleteModalOpened(false)}
        title="Delete User"
      >
        <Stack>
          {selectedUser && (
            <>
              <Text>
                Are you sure you want to delete this user?
              </Text>
              <Text fw={500}>
                {selectedUser.first_name} {selectedUser.last_name} ({selectedUser.email})
              </Text>
              <Text size="sm" c="red">
                This action cannot be undone. All user data will be permanently deleted.
              </Text>
            </>
          )}

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={() => setDeleteModalOpened(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleDeleteUser}
              loading={actionLoading}
              color="red"
            >
              Delete User
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
};
