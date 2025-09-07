"use client";

import { useState, useEffect } from "react";
import { useRouter, useParams } from "next/navigation";
import { useSession } from "@/lib/auth-client";
import {
  Card,
  CardBody,
  CardHeader,
  Button,
  Chip,
  Avatar,
  Progress,
  Spinner,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Input,
  Divider,
} from "@nextui-org/react";
import {
  ArrowLeft,
  Users,
  Clock,
  MapPin,
  FileText,
  Upload,
  Download,
  Timer,
  Share2,
  Settings,
  Copy,
  CheckCircle,
  AlertCircle,
  Building2,
} from "lucide-react";

interface Organization {
  id: string;
  name: string;
  logoUrl?: string;
  brandColor?: string;
}

interface GroupMember {
  userId: string;
  role: "creator" | "member";
  joinedAt: string;
  user: {
    id: string;
    name?: string;
    email: string;
    image?: string;
  };
}

interface GroupFile {
  id: string;
  filename: string;
  originalName: string;
  mimetype: string;
  size: number;
  path: string;
  uploaderId?: string;
  uploader?: {
    name?: string;
    email: string;
  };
  createdAt: string;
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
  isExpired: boolean;
  createdAt: string;
  members: GroupMember[];
  files: GroupFile[];
  code: string;
}

export default function GroupDetailsPage() {
  const router = useRouter();
  const params = useParams();
  const groupId = params?.groupId as string;
  const { data: session } = useSession();
  const [group, setGroup] = useState<Group | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showExtendModal, setShowExtendModal] = useState(false);
  const [extending, setExtending] = useState(false);
  const [copied, setCopied] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  useEffect(() => {
    if (groupId) {
      fetchGroup();
    }
  }, [groupId]);

  const fetchGroup = async () => {
    try {
      const response = await fetch(`/api/groups/${groupId}`);
      if (!response.ok) {
        throw new Error("Group not found");
      }
      const data = await response.json();
      setGroup(data);
    } catch (err: any) {
      setError(err.message || "Failed to load group");
    } finally {
      setLoading(false);
    }
  };

  const handleExtendTime = async () => {
    setExtending(true);
    try {
      const response = await fetch(`/api/groups/${groupId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "extend" }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to extend time");
      }

      await fetchGroup();
      setShowExtendModal(false);
    } catch (err: any) {
      alert(err.message);
    } finally {
      setExtending(false);
    }
  };

  const handleCopyCode = () => {
    if (group?.code) {
      navigator.clipboard.writeText(group.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      handleFileUpload(file);
    }
  };

  const handleFileUpload = async (file: File) => {
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);

      const response = await fetch(`/api/groups/${groupId}/files`, {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to upload file");
      }

      // Refresh the group data to show the new file
      await fetchGroup();
      setSelectedFile(null);
    } catch (err: any) {
      alert(err.message || "Failed to upload file");
    } finally {
      setUploading(false);
    }
  };

  const handleFileDownload = (file: GroupFile) => {
    // Create a download link
    const link = document.createElement("a");
    link.href = file.path;
    link.download = file.originalName;
    link.target = "_blank";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / (1024 * 1024)).toFixed(1) + " MB";
  };

  const getTimeRemaining = () => {
    if (!group) return { text: "Loading...", percentage: 0 };
    
    const now = new Date().getTime();
    const created = new Date(group.createdAt).getTime();
    const expiry = new Date(group.expiresAt).getTime();
    const total = expiry - created;
    const elapsed = now - created;
    const remaining = expiry - now;

    if (remaining <= 0) {
      return { text: "Expired", percentage: 100, expired: true };
    }

    const hours = Math.floor(remaining / (1000 * 60 * 60));
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
    const percentage = (elapsed / total) * 100;

    const text = hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
    return { text, percentage, expired: false };
  };

  const timeInfo = getTimeRemaining();
  const isCreator = group?.creatorId === session?.user?.id || 
                   group?.members?.some(m => m.userId === session?.user?.id && m.role === "creator");

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  if (error || !group) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Card className="max-w-md">
          <CardBody className="text-center py-8">
            <AlertCircle className="w-12 h-12 text-danger mx-auto mb-4" />
            <h2 className="text-xl font-semibold mb-2">Group Not Found</h2>
            <p className="text-gray-500 mb-4">{error || "This group doesn't exist or has been deleted."}</p>
            <Button color="primary" onPress={() => router.push("/dashboard")}>
              Back to Dashboard
            </Button>
          </CardBody>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      {/* Header */}
      <div 
        className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-md shadow-sm border-b dark:border-gray-700"
        style={{
          borderTop: group.organization?.brandColor ? `4px solid ${group.organization.brandColor}` : undefined,
        }}
      >
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center gap-4">
              <Button
                variant="light"
                isIconOnly
                onPress={() => router.push("/dashboard")}
              >
                <ArrowLeft className="w-5 h-5" />
              </Button>
              {group.organization && (
                <Avatar
                  src={group.organization.logoUrl}
                  size="md"
                  className="border-2"
                  style={{ borderColor: group.organization.brandColor }}
                />
              )}
              <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                  {group.name}
                </h1>
                {group.organization && (
                  <p className="text-sm text-gray-500 dark:text-gray-400 flex items-center gap-1">
                    <Building2 className="w-3 h-3" />
                    {group.organization.name}
                  </p>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2">
              {isCreator && (
                <Button
                  variant="light"
                  isIconOnly
                >
                  <Settings className="w-5 h-5" />
                </Button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Status Bar */}
        <Card className="mb-6">
          <CardBody>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-6">
                <Chip
                  color={group.isActive && !timeInfo.expired ? "success" : "danger"}
                  variant="flat"
                  startContent={timeInfo.expired ? <AlertCircle className="w-4 h-4" /> : <CheckCircle className="w-4 h-4" />}
                >
                  {group.isActive && !timeInfo.expired ? "Active" : "Expired"}
                </Chip>
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <MapPin className="w-4 h-4" />
                  <span>{group.radius}m radius</span>
                </div>
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <Users className="w-4 h-4" />
                  <span>{group.members.length} members</span>
                </div>
                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <FileText className="w-4 h-4" />
                  <span>{group.files.length} files</span>
                </div>
              </div>
            </div>

            {/* Time Progress */}
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <div className="flex items-center gap-2">
                  <Clock className="w-4 h-4 text-gray-500" />
                  <span className="text-sm font-medium">Time Remaining: {timeInfo.text}</span>
                </div>
                {isCreator && !timeInfo.expired && group.extendedCount < 3 && (
                  <Button
                    size="sm"
                    variant="flat"
                    onPress={() => setShowExtendModal(true)}
                    startContent={<Timer className="w-4 h-4" />}
                  >
                    Extend ({3 - group.extendedCount} left)
                  </Button>
                )}
              </div>
              <Progress
                value={timeInfo.percentage}
                color={timeInfo.expired ? "danger" : timeInfo.percentage > 75 ? "warning" : "primary"}
                className="max-w-full"
              />
            </div>
          </CardBody>
        </Card>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main Content Area */}
          <div className="lg:col-span-2 space-y-6">
            {/* Description */}
            {group.description && (
              <Card>
                <CardHeader>
                  <h2 className="text-lg font-semibold">About</h2>
                </CardHeader>
                <CardBody>
                  <p className="text-gray-600 dark:text-gray-400">{group.description}</p>
                </CardBody>
              </Card>
            )}

            {/* Files */}
            <Card>
              <CardHeader className="flex justify-between">
                <h2 className="text-lg font-semibold">Files</h2>
                <div className="flex items-center gap-2">
                  <input
                    type="file"
                    id="file-upload"
                    className="hidden"
                    onChange={handleFileSelect}
                    disabled={timeInfo.expired || uploading}
                  />
                  <Button
                    color="primary"
                    size="sm"
                    startContent={<Upload className="w-4 h-4" />}
                    isDisabled={timeInfo.expired || uploading}
                    isLoading={uploading}
                    onPress={() => document.getElementById("file-upload")?.click()}
                  >
                    {uploading ? "Uploading..." : "Upload File"}
                  </Button>
                </div>
              </CardHeader>
              <CardBody>
                {group.files.length === 0 ? (
                  <div className="text-center py-8">
                    <FileText className="w-12 h-12 text-gray-300 mx-auto mb-3" />
                    <p className="text-gray-500">No files shared yet</p>
                    <p className="text-sm text-gray-400 mt-1">Upload files to share with the group</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {group.files.map((file) => (
                      <div
                        key={file.id}
                        className="flex items-center justify-between p-3 rounded-lg border hover:bg-gray-50 dark:hover:bg-gray-800"
                      >
                        <div className="flex items-center gap-3">
                          <FileText className="w-5 h-5 text-gray-400" />
                          <div>
                            <p className="font-medium">{file.originalName}</p>
                            <p className="text-xs text-gray-500">
                              {formatFileSize(file.size)} • Uploaded by {file.uploader?.name || file.uploader?.email || "Anonymous"}
                            </p>
                          </div>
                        </div>
                        <Button
                          size="sm"
                          variant="light"
                          isIconOnly
                          onPress={() => handleFileDownload(file)}
                        >
                          <Download className="w-4 h-4" />
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </CardBody>
            </Card>
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Members */}
            <Card>
              <CardHeader>
                <h2 className="text-lg font-semibold">Members</h2>
              </CardHeader>
              <CardBody className="space-y-3">
                {group.members.map((member) => (
                  <div key={member.userId} className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Avatar
                        src={member.user.image}
                        name={member.user.name || member.user.email}
                        size="sm"
                      />
                      <div>
                        <p className="text-sm font-medium">
                          {member.user.name || member.user.email}
                        </p>
                        <p className="text-xs text-gray-500">
                          {member.role === "creator" ? "Creator" : "Member"}
                        </p>
                      </div>
                    </div>
                    {member.role === "creator" && (
                      <Chip size="sm" color="primary" variant="flat">
                        Creator
                      </Chip>
                    )}
                  </div>
                ))}
              </CardBody>
            </Card>

            {/* Location Info */}
            <Card>
              <CardHeader>
                <h2 className="text-lg font-semibold">Location</h2>
              </CardHeader>
              <CardBody>
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <MapPin className="w-4 h-4 text-gray-500" />
                    <span className="text-sm">
                      {group.latitude.toFixed(4)}, {group.longitude.toFixed(4)}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Users className="w-4 h-4 text-gray-500" />
                    <span className="text-sm">
                      {group.radius}m sharing radius
                    </span>
                  </div>
                </div>
              </CardBody>
            </Card>
          </div>
        </div>
      </div>

      {/* Extend Time Modal */}
      <Modal isOpen={showExtendModal} onClose={() => setShowExtendModal(false)}>
        <ModalContent>
          <ModalHeader>Extend Group Time</ModalHeader>
          <ModalBody>
            <p className="text-sm text-gray-600 dark:text-gray-300">
              Extend the group expiration by 4 hours?
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">
              You have {3 - group.extendedCount} extension{3 - group.extendedCount !== 1 ? "s" : ""} remaining.
            </p>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={() => setShowExtendModal(false)}>
              Cancel
            </Button>
            <Button
              color="primary"
              onPress={handleExtendTime}
              isLoading={extending}
            >
              Extend by 4 Hours
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}