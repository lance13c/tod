'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { 
  Card, 
  CardBody, 
  CardHeader, 
  Button, 
  Chip, 
  Avatar, 
  Progress,
  Divider,
  Input,
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  useDisclosure
} from '@nextui-org/react';
import { 
  Users, 
  Clock, 
  MapPin, 
  Copy, 
  Share2, 
  Upload, 
  Download,
  File,
  Image,
  Film,
  FileText,
  LogOut,
  CheckCircle,
  AlertCircle,
  Wifi,
  WifiOff
} from 'lucide-react';
import { api } from '@/lib/trpc/client';
import { LocationPermission } from '@/components/location/LocationPermission';

interface Participant {
  id: string;
  nickname?: string | null;
  isConnected: boolean;
  user?: {
    id: string;
    name: string | null;
    image: string | null;
  } | null;
  guest?: {
    id: string;
    nickname: string | null;
  } | null;
}

function getDeviceFingerprint(): string {
  if (typeof window === 'undefined') return '';
  const stored = localStorage.getItem('device_fingerprint');
  if (stored) return stored;
  const fingerprint = Math.random().toString(36).substring(2) + Date.now().toString(36);
  localStorage.setItem('device_fingerprint', fingerprint);
  return fingerprint;
}

export default function SessionPage() {
  const params = useParams();
  const router = useRouter();
  const code = params.code as string;
  
  const [hasJoined, setHasJoined] = useState(false);
  const [participantId, setParticipantId] = useState<string | null>(null);
  const [peerId, setPeerId] = useState<string | null>(null);
  const [location, setLocation] = useState<{ latitude: number; longitude: number } | null>(null);
  const [nickname, setNickname] = useState('');
  const [copied, setCopied] = useState(false);
  const { isOpen, onOpen, onClose } = useDisclosure();

  // Fetch session data
  const { data: session, refetch } = api.shareSession.getSession.useQuery(
    { code: code.toUpperCase() },
    { 
      enabled: !!code,
      refetchInterval: 5000, // Poll every 5 seconds for updates
    }
  );

  // Join session mutation
  const joinSession = api.shareSession.joinSession.useMutation();

  // Handle successful join
  useEffect(() => {
    if (joinSession.isSuccess && joinSession.data) {
      setHasJoined(true);
      setParticipantId(joinSession.data.participant.id);
      setPeerId(joinSession.data.peerId);
      onClose();
    }
  }, [joinSession.isSuccess, joinSession.data, onClose]);

  // Leave session mutation
  const leaveSession = api.shareSession.leaveSession.useMutation();

  // Handle successful leave
  useEffect(() => {
    if (leaveSession.isSuccess) {
      router.push('/share');
    }
  }, [leaveSession.isSuccess, router]);

  // Ping mutation to maintain connection
  const ping = api.shareSession.ping.useMutation();

  // Send ping every 30 seconds to maintain connection
  useEffect(() => {
    if (!participantId) return;

    const interval = setInterval(() => {
      ping.mutate({ participantId });
    }, 30000);

    return () => clearInterval(interval);
  }, [participantId]);

  const handleLocationGranted = (latitude: number, longitude: number) => {
    setLocation({ latitude, longitude });
    if (nickname) {
      handleJoinSession(latitude, longitude);
    } else {
      onOpen(); // Show nickname modal
    }
  };

  const handleJoinSession = (lat?: number, lon?: number) => {
    const finalLocation = lat && lon ? { latitude: lat, longitude: lon } : location;
    if (!finalLocation) return;

    joinSession.mutate({
      code: code.toUpperCase(),
      latitude: finalLocation.latitude,
      longitude: finalLocation.longitude,
      nickname: nickname || undefined,
      isGuest: true,
      guestFingerprint: getDeviceFingerprint(),
    });
  };

  const handleLeaveSession = () => {
    if (participantId && session) {
      leaveSession.mutate({
        sessionId: session.id,
        participantId,
      });
    }
  };

  const copyCode = () => {
    navigator.clipboard.writeText(code.toUpperCase());
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const getFileIcon = (mimetype: string) => {
    if (mimetype.startsWith('image/')) return <Image className="w-5 h-5" />;
    if (mimetype.startsWith('video/')) return <Film className="w-5 h-5" />;
    if (mimetype.includes('pdf')) return <FileText className="w-5 h-5" />;
    return <File className="w-5 h-5" />;
  };

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  };

  if (!session) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-8">
            <AlertCircle className="w-16 h-16 text-warning mx-auto mb-4" />
            <h3 className="text-xl font-semibold mb-2">Session Not Found</h3>
            <p className="text-default-500 mb-4">
              The session code "{code}" doesn't exist or has expired.
            </p>
            <Button color="primary" onPress={() => router.push('/share')}>
              Create New Session
            </Button>
          </CardBody>
        </Card>
      </div>
    );
  }

  // Check if session has expired
  const isExpired = new Date(session.expiresAt) < new Date();
  if (isExpired) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-8">
            <Clock className="w-16 h-16 text-default-400 mx-auto mb-4" />
            <h3 className="text-xl font-semibold mb-2">Session Expired</h3>
            <p className="text-default-500 mb-4">
              This session has ended. Create a new one to continue sharing.
            </p>
            <Button color="primary" onPress={() => router.push('/share')}>
              Create New Session
            </Button>
          </CardBody>
        </Card>
      </div>
    );
  }

  // Show location permission if not joined
  if (!hasJoined && !location) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <LocationPermission
          onLocationGranted={handleLocationGranted}
          title={`Join "${session.name}"`}
          description="We need to verify you're nearby to join this session"
        />
      </div>
    );
  }

  // Show nickname input modal
  if (!hasJoined && location) {
    return (
      <>
        <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
          <Card className="max-w-md w-full">
            <CardHeader className="flex flex-col gap-1 pb-2">
              <h2 className="text-2xl font-bold">Join Session</h2>
              <p className="text-default-500">Enter a nickname to identify yourself</p>
            </CardHeader>
            <CardBody className="gap-4">
              <div className="bg-default-100 rounded-lg p-3">
                <div className="flex items-center gap-2 mb-1">
                  <Share2 className="w-4 h-4 text-primary" />
                  <span className="font-semibold">{session.name}</span>
                </div>
                {session.description && (
                  <p className="text-sm text-default-500">{session.description}</p>
                )}
              </div>

              <Input
                label="Your Nickname (Optional)"
                placeholder="e.g., John"
                value={nickname}
                onValueChange={setNickname}
                description="How others will see you in the session"
              />

              <div className="flex gap-3">
                <Button variant="flat" onPress={() => router.push('/share')}>
                  Cancel
                </Button>
                <Button
                  color="primary"
                  onPress={() => handleJoinSession()}
                  isLoading={joinSession.isPending}
                  className="flex-1"
                >
                  Join as {nickname || 'Guest'}
                </Button>
              </div>
            </CardBody>
          </Card>
        </div>
      </>
    );
  }

  // Main session view
  const timeRemaining = Math.max(0, new Date(session.expiresAt).getTime() - Date.now());
  const hoursRemaining = Math.floor(timeRemaining / (1000 * 60 * 60));
  const minutesRemaining = Math.floor((timeRemaining % (1000 * 60 * 60)) / (1000 * 60));

  return (
    <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <Card className="mb-4">
          <CardBody className="flex flex-row items-center justify-between">
            <div className="flex-1">
              <h1 className="text-2xl font-bold mb-1">{session.name}</h1>
              {session.description && (
                <p className="text-default-500">{session.description}</p>
              )}
              <div className="flex flex-wrap gap-3 mt-3">
                <Chip
                  startContent={<Users className="w-3 h-3" />}
                  size="sm"
                  variant="flat"
                >
                  {session.participants.length}/{session.maxParticipants}
                </Chip>
                <Chip
                  startContent={<Clock className="w-3 h-3" />}
                  size="sm"
                  variant="flat"
                  color={hoursRemaining < 1 ? 'warning' : 'default'}
                >
                  {hoursRemaining}h {minutesRemaining}m remaining
                </Chip>
                <Chip
                  startContent={<MapPin className="w-3 h-3" />}
                  size="sm"
                  variant="flat"
                >
                  {session.geoLockRadius}m radius
                </Chip>
              </div>
            </div>
            <div className="flex gap-2">
              <Button
                variant="flat"
                onPress={copyCode}
                startContent={copied ? <CheckCircle className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
              >
                {copied ? 'Copied!' : code.toUpperCase()}
              </Button>
              <Button
                color="danger"
                variant="flat"
                onPress={handleLeaveSession}
                startContent={<LogOut className="w-4 h-4" />}
              >
                Leave
              </Button>
            </div>
          </CardBody>
        </Card>

        <div className="grid md:grid-cols-3 gap-4">
          {/* Participants */}
          <Card className="md:col-span-1">
            <CardHeader>
              <h3 className="font-semibold">Participants ({session.participants.length})</h3>
            </CardHeader>
            <CardBody className="gap-2">
              {session.participants.map((participant) => {
                const displayName = participant.nickname || 
                                   participant.user?.name || 
                                   participant.guest?.nickname || 
                                   'Guest';
                const avatar = participant.user?.image;

                return (
                  <div key={participant.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-default-100">
                    <Avatar
                      src={avatar || undefined}
                      name={displayName}
                      size="sm"
                    />
                    <div className="flex-1">
                      <p className="text-sm font-medium">{displayName}</p>
                    </div>
                    <Chip
                      size="sm"
                      variant="dot"
                      color={participant.isConnected ? 'success' : 'default'}
                      startContent={participant.isConnected ? <Wifi className="w-3 h-3" /> : <WifiOff className="w-3 h-3" />}
                    >
                      {participant.isConnected ? 'Connected' : 'Offline'}
                    </Chip>
                  </div>
                );
              })}
            </CardBody>
          </Card>

          {/* Files Area */}
          <Card className="md:col-span-2">
            <CardHeader className="flex justify-between">
              <h3 className="font-semibold">Shared Files ({session._count.documents})</h3>
              <Button
                color="primary"
                size="sm"
                startContent={<Upload className="w-4 h-4" />}
              >
                Upload Files
              </Button>
            </CardHeader>
            <CardBody>
              {session.documents.length === 0 ? (
                <div className="text-center py-12">
                  <Upload className="w-16 h-16 text-default-300 mx-auto mb-4" />
                  <p className="text-default-500">No files shared yet</p>
                  <p className="text-sm text-default-400 mt-1">
                    Upload files to share with participants
                  </p>
                </div>
              ) : (
                <div className="grid gap-2">
                  {session.documents.map((doc) => (
                    <div key={doc.id} className="flex items-center gap-3 p-3 rounded-lg bg-default-50 hover:bg-default-100">
                      <div className="flex-shrink-0">
                        {getFileIcon(doc.mimetype)}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="font-medium truncate">{doc.originalName}</p>
                        <p className="text-sm text-default-500">
                          {formatFileSize(doc.size)}
                        </p>
                      </div>
                      {doc.transferStatus === 'transferring' && (
                        <Progress
                          size="sm"
                          value={doc.transferProgress}
                          className="w-20"
                        />
                      )}
                      {doc.transferStatus === 'completed' && (
                        <Button
                          size="sm"
                          variant="flat"
                          startContent={<Download className="w-4 h-4" />}
                        >
                          Download
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardBody>
          </Card>
        </div>
      </div>
    </div>
  );
}