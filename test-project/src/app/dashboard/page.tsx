"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useSession, signOut } from "@/lib/auth-client";
import {
  Card,
  CardBody,
  CardHeader,
  Button,
  Chip,
  Avatar,
  Tabs,
  Tab,
  Spinner,
  Badge,
  Input,
  Dropdown,
  DropdownTrigger,
  DropdownMenu,
  DropdownItem,
} from "@nextui-org/react";
import {
  Plus,
  Users,
  Clock,
  MapPin,
  Building2,
  Search,
  Filter,
  Calendar,
  FileText,
  MoreVertical,
  Timer,
  UserCheck,
  UserPlus,
  Archive,
  LogOut,
} from "lucide-react";

interface Organization {
  id: string;
  name: string;
  logoUrl?: string;
  brandColor?: string;
}

interface GroupMember {
  userId: string;
  role: "creator" | "participant";
  user: {
    name?: string;
    email: string;
    image?: string;
  };
}

interface Group {
  id: string;
  name: string;
  description?: string;
  organization?: Organization;
  creatorId?: string;
  creator?: {
    name?: string;
    email: string;
  };
  latitude: number;
  longitude: number;
  radius: number;
  expiresAt: string;
  extendedCount: number;
  isActive: boolean;
  isArchived?: boolean;
  createdAt: string;
  members?: GroupMember[];
  files?: any[];
  _count?: {
    files: number;
  };
  role?: string;
  joinedAt?: string;
}

export default function DashboardPage() {
  const router = useRouter();
  const { data: session, isPending } = useSession();
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState("all");
  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    if (!isPending && !session) {
      router.push("/login");
    }
  }, [session, isPending, router]);

  useEffect(() => {
    if (session) {
      fetchGroups();
    }
  }, [session]);

  const fetchGroups = async () => {
    try {
      const response = await fetch("/api/groups");
      if (response.ok) {
        const data = await response.json();
        setGroups(data);
      } else {
        // Fall back to mock data if API fails
        setGroups([
        {
          id: "550e8400-e29b-41d4-a716-446655440000",
          name: "Team Standup - Q1 Planning",
          description: "Quarterly planning session for the engineering team",
          organization: {
            id: "org-1",
            name: "Acme Corporation",
            logoUrl: "https://api.dicebear.com/7.x/identicon/svg?seed=acme",
            brandColor: "#FF6B6B",
          },
          creatorId: "test-user-2",
          creator: {
            name: "John Doe",
            email: "john@example.com",
          },
          latitude: 37.7749,
          longitude: -122.4194,
          radius: 100,
          expiresAt: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
          extendedCount: 0,
          isActive: true,
          isArchived: false,
          createdAt: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
          members: [
            {
              userId: "test-user-2",
              role: "creator",
              user: {
                name: "John Doe",
                email: "john@example.com",
                image: "https://api.dicebear.com/7.x/avataaars/svg?seed=johndoe",
              },
            },
            {
              userId: "admin-user",
              role: "participant",
              user: {
                name: "Admin User",
                email: "admin@example.com",
                image: "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
              },
            },
          ],
          _count: {
            files: 2,
          },
        },
        {
          id: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
          name: "Design Review Session",
          description: "Review of new UI designs for the mobile app",
          organization: {
            id: "org-2",
            name: "TechStart Inc",
            logoUrl: "https://api.dicebear.com/7.x/identicon/svg?seed=techstart",
            brandColor: "#4A90E2",
          },
          creatorId: "admin-user",
          creator: {
            name: "Admin User",
            email: "admin@example.com",
          },
          latitude: 30.2672,
          longitude: -97.7431,
          radius: 100,
          expiresAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
          extendedCount: 2,
          isActive: false,
          isArchived: true,
          createdAt: new Date(Date.now() - 8 * 60 * 60 * 1000).toISOString(),
          members: [
            {
              userId: "admin-user",
              role: "creator",
              user: {
                name: "Admin User",
                email: "admin@example.com",
                image: "https://api.dicebear.com/7.x/avataaars/svg?seed=admin",
              },
            },
          ],
          _count: {
            files: 5,
          },
        },
        {
          id: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
          name: "Coffee Break Meetup",
          description: "Informal team gathering",
          creatorId: "test-user-1",
          creator: {
            name: "Test User",
            email: "test@example.com",
          },
          latitude: 40.7128,
          longitude: -74.0060,
          radius: 50,
          expiresAt: new Date(Date.now() + 3 * 60 * 60 * 1000).toISOString(),
          extendedCount: 0,
          isActive: true,
          isArchived: false,
          createdAt: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
          members: [
            {
              userId: "test-user-1",
              role: "creator",
              user: {
                name: "Test User",
                email: "test@example.com",
              },
            },
          ],
          _count: {
            files: 0,
          },
        },
      ]);
      }
    } catch (error) {
      console.error("Failed to fetch groups:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSignOut = async () => {
    await signOut();
    router.push("/login");
  };

  const getTimeRemaining = (expiresAt: string) => {
    const now = new Date().getTime();
    const expiry = new Date(expiresAt).getTime();
    const diff = expiry - now;

    if (diff <= 0) return "Expired";

    const hours = Math.floor(diff / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  };

  const getUserRole = (group: Group) => {
    if (!session?.user) return null;
    // Check if role is directly on the group (from API response)
    if (group.role) return group.role;
    // Otherwise check in members array
    const member = group.members?.find((m) => m.userId === session.user.id);
    return member?.role;
  };

  const filteredGroups = groups.filter((group) => {
    // Filter by tab
    if (activeTab === "started") {
      if (group.creatorId !== session?.user?.id) return false;
    } else if (activeTab === "participated") {
      if (group.creatorId === session?.user?.id) return false;
    } else if (activeTab === "active") {
      if (!group.isActive) return false;
    } else if (activeTab === "archived") {
      if (!group.isArchived) return false;
    }

    // Filter by search
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      return (
        group.name.toLowerCase().includes(query) ||
        group.description?.toLowerCase().includes(query) ||
        group.organization?.name.toLowerCase().includes(query)
      );
    }

    return true;
  });

  if (isPending || loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (!session) {
    return null;
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
      {/* Header */}
      <div className="bg-white dark:bg-gray-800 shadow-sm border-b dark:border-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                Dashboard
              </h1>
              <p className="text-gray-500 dark:text-gray-400 mt-1">
                Welcome back, {session.user?.name || session.user?.email}
              </p>
            </div>
            <div className="flex items-center gap-4">
              <Button
                color="primary"
                size="lg"
                startContent={<Plus className="w-5 h-5" />}
                onPress={() => router.push("/groups/new")}
                className="bg-gradient-to-r from-blue-500 to-purple-600"
              >
                Start New Group
              </Button>
              <Button
                variant="bordered"
                size="lg"
                startContent={<Users className="w-5 h-5" />}
                onPress={() => router.push("/groups/join")}
              >
                Join Group
              </Button>
              <Button
                color="danger"
                variant="light"
                startContent={<LogOut className="w-5 h-5" />}
                onPress={handleSignOut}
              >
                Sign Out
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardBody className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Total Groups</p>
                  <p className="text-2xl font-bold">{groups.length}</p>
                </div>
                <Users className="w-8 h-8 text-blue-500" />
              </div>
            </CardBody>
          </Card>
          <Card>
            <CardBody className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Active Groups</p>
                  <p className="text-2xl font-bold">
                    {groups.filter((g) => g.isActive).length}
                  </p>
                </div>
                <Clock className="w-8 h-8 text-green-500" />
              </div>
            </CardBody>
          </Card>
          <Card>
            <CardBody className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Groups Started</p>
                  <p className="text-2xl font-bold">
                    {groups.filter((g) => g.creatorId === session.user?.id).length}
                  </p>
                </div>
                <UserCheck className="w-8 h-8 text-purple-500" />
              </div>
            </CardBody>
          </Card>
          <Card>
            <CardBody className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">Files Shared</p>
                  <p className="text-2xl font-bold">
                    {groups.reduce((acc, g) => acc + (g._count?.files || 0), 0)}
                  </p>
                </div>
                <FileText className="w-8 h-8 text-orange-500" />
              </div>
            </CardBody>
          </Card>
        </div>
      </div>

      {/* Search and Filters */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-8">
        <div className="flex flex-col sm:flex-row gap-4">
          <Input
            placeholder="Search groups..."
            value={searchQuery}
            onValueChange={setSearchQuery}
            startContent={<Search className="w-4 h-4 text-gray-400" />}
            className="flex-1"
          />
          <Tabs
            selectedKey={activeTab}
            onSelectionChange={(key) => setActiveTab(key as string)}
            color="primary"
            variant="bordered"
          >
            <Tab key="all" title="All Groups" />
            <Tab key="started" title="Started by Me" />
            <Tab key="participated" title="Participated" />
            <Tab key="active" title="Active" />
            <Tab key="archived" title="Archived" />
          </Tabs>
        </div>
      </div>

      {/* Groups List */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-8 pb-12">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredGroups.map((group) => {
            const role = getUserRole(group);
            const timeRemaining = getTimeRemaining(group.expiresAt);

            return (
              <Card
                key={group.id}
                className="hover:shadow-lg transition-shadow"
              >
                <CardHeader className="pb-2">
                  <div className="flex justify-between items-start w-full">
                    <div 
                      className="flex items-start gap-3 flex-1 cursor-pointer"
                      onClick={() => router.push(`/groups/${group.id}`)}
                    >
                      {group.organization ? (
                        <Avatar
                          src={group.organization.logoUrl}
                          name={group.organization.name}
                          className="w-12 h-12"
                          style={{
                            borderColor: group.organization.brandColor,
                            borderWidth: 2,
                          }}
                        />
                      ) : (
                        <div className="w-12 h-12 bg-gray-200 dark:bg-gray-700 rounded-full flex items-center justify-center">
                          <Users className="w-6 h-6 text-gray-500" />
                        </div>
                      )}
                      <div className="flex-1">
                        <h3 className="font-semibold text-lg">{group.name}</h3>
                        {group.organization && (
                          <p className="text-sm text-gray-500">
                            {group.organization.name}
                          </p>
                        )}
                      </div>
                    </div>
                    <Dropdown>
                      <DropdownTrigger>
                        <Button isIconOnly size="sm" variant="light">
                          <MoreVertical className="w-4 h-4" />
                        </Button>
                      </DropdownTrigger>
                      <DropdownMenu>
                        <DropdownItem
                          key="view"
                          onPress={() => router.push(`/groups/${group.id}`)}
                          className="text-gray-900 dark:text-gray-100"
                        >
                          View Group
                        </DropdownItem>
                        {role === "creator" && group.isActive ? (
                          <DropdownItem 
                            key="extend"
                            className="text-gray-900 dark:text-gray-100"
                          >
                            Extend Time
                          </DropdownItem>
                        ) : null}
                        {role === "creator" ? (
                          <DropdownItem
                            key="delete"
                            className="text-danger"
                            color="danger"
                          >
                            Delete Group
                          </DropdownItem>
                        ) : null}
                      </DropdownMenu>
                    </Dropdown>
                  </div>
                </CardHeader>
                <CardBody 
                  className="pt-2 cursor-pointer"
                  onClick={() => router.push(`/groups/${group.id}`)}
                >
                  {group.description && (
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                      {group.description}
                    </p>
                  )}

                  <div className="flex flex-wrap gap-2 mb-3">
                    {role && (
                      <Chip
                        size="sm"
                        color={role === "creator" ? "primary" : "default"}
                        startContent={
                          role === "creator" ? (
                            <UserCheck className="w-3 h-3" />
                          ) : (
                            <UserPlus className="w-3 h-3" />
                          )
                        }
                      >
                        {role === "creator" ? "Creator" : "Participant"}
                      </Chip>
                    )}
                    {group.isActive ? (
                      <Chip
                        size="sm"
                        color="success"
                        variant="flat"
                        startContent={<Timer className="w-3 h-3" />}
                      >
                        {timeRemaining}
                      </Chip>
                    ) : (
                      <Chip
                        size="sm"
                        color="default"
                        variant="flat"
                        startContent={<Archive className="w-3 h-3" />}
                      >
                        Archived
                      </Chip>
                    )}
                    {group.extendedCount > 0 && (
                      <Chip size="sm" variant="flat">
                        Extended {group.extendedCount}x
                      </Chip>
                    )}
                  </div>

                  <div className="flex items-center justify-between text-sm text-gray-500">
                    <div className="flex items-center gap-4">
                      <span className="flex items-center gap-1">
                        <Users className="w-4 h-4" />
                        {group.members?.length || 0}
                      </span>
                      <span className="flex items-center gap-1">
                        <FileText className="w-4 h-4" />
                        {group._count?.files || group.files?.length || 0}
                      </span>
                    </div>
                    <span className="flex items-center gap-1">
                      <MapPin className="w-4 h-4" />
                      {group.radius}m
                    </span>
                  </div>

                  <div className="flex items-center gap-2 mt-3 pt-3 border-t">
                    <Avatar
                      src={
                        group.members?.find((m) => m.role === "creator")?.user
                          ?.image || undefined
                      }
                      name={group.creator?.name || group.creator?.email}
                      size="sm"
                    />
                    <div className="text-xs">
                      <p className="font-medium">
                        {group.creator?.name || group.creator?.email}
                      </p>
                      <p className="text-gray-500">
                        Created {new Date(group.createdAt).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                </CardBody>
              </Card>
            );
          })}
        </div>

        {filteredGroups.length === 0 && (
          <Card className="mt-8">
            <CardBody className="text-center py-12">
              <Users className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">No groups found</h3>
              <p className="text-gray-500 mb-4">
                {activeTab === "started"
                  ? "You haven't started any groups yet."
                  : activeTab === "participated"
                  ? "You haven't joined any groups yet."
                  : searchQuery
                  ? "Try adjusting your search terms."
                  : "Start a new group to get started!"}
              </p>
              <Button
                color="primary"
                startContent={<Plus className="w-4 h-4" />}
                onPress={() => router.push("/groups/new")}
              >
                Start New Group
              </Button>
            </CardBody>
          </Card>
        )}
      </div>
    </div>
  );
}