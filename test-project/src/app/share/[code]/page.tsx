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
  useDisclosure,
  Spinner
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
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const { isOpen, onOpen, onClose } = useDisclosure();

  // Fetch session data
  const { data: session, refetch, isLoading: sessionLoading } = api.shareSession.getSession.useQuery(
    { code: code.toUpperCase() },
    { 
      enabled: !!code,
      refetchInterval: 5000, // Poll every 5 seconds for updates
    }
  );

  // Join session mutation
  const joinSession = api.shareSession.joinSession.useMutation({
    onError: (error) => {
      // If already in session, just mark as joined (not really an error)
      if (error.message.includes('already in this session')) {
        console.log('User already in session, rejoining...');
        // Check localStorage for existing participant ID
        if (session) {
          const storageKey = `session_participant_${session.id}`;
          const storedParticipantId = localStorage.getItem(storageKey);
          
          if (storedParticipantId) {
            const existingParticipant = session.participants?.find(
              p => p.id === storedParticipantId
            );
            
            if (existingParticipant) {
              console.log('Rejoining with existing participant ID');
              setHasJoined(true);
              setParticipantId(existingParticipant.id);
              return;
            }
          }
          
          // Just mark as joined even if we can't find the exact participant
          // The UI will still work and show them as a participant
          console.log('Marking user as joined (existing participant)');
          setHasJoined(true);
          // Try to find any guest participant that might be us
          const possibleParticipant = session.participants?.find(p => p.guest);
          if (possibleParticipant) {
            setParticipantId(possibleParticipant.id);
            localStorage.setItem(storageKey, possibleParticipant.id);
          }
        }
      } else {
        // Only show alert for actual errors
        console.error('Failed to join session:', error);
        alert(`Failed to join session: ${error.message}`);
      }
    }
  });

  // Handle successful join
  useEffect(() => {
    if (joinSession.isSuccess && joinSession.data) {
      setHasJoined(true);
      setParticipantId(joinSession.data.participant.id);
      setPeerId(joinSession.data.peerId);
      // Store participant ID for auto-rejoin
      if (session) {
        localStorage.setItem(`session_participant_${session.id}`, joinSession.data.participant.id);
      }
      onClose();
    }
  }, [joinSession.isSuccess, joinSession.data, session, onClose]);

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

  // Check if already a participant when session loads
  useEffect(() => {
    if (session && !hasJoined) {
      // Check localStorage for existing participant ID for this session
      const storageKey = `session_participant_${session.id}`;
      const storedParticipantId = localStorage.getItem(storageKey);
      
      if (storedParticipantId) {
        const existingParticipant = session.participants?.find(
          p => p.id === storedParticipantId
        );
        
        if (existingParticipant) {
          console.log('Found existing participant, rejoining automatically');
          setHasJoined(true);
          setParticipantId(existingParticipant.id);
        }
      }
    }
  }, [session, hasJoined]);

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
    // The UI will automatically show the nickname input screen
    // when location is set and hasJoined is false
  };

  const handleJoinSession = (lat?: number, lon?: number) => {
    const finalLocation = lat && lon ? { latitude: lat, longitude: lon } : location;
    if (!finalLocation) {
      console.error('No location available for joining session');
      alert('Location is required to join the session. Please enable location access.');
      return;
    }

    const payload = {
      code: code.toUpperCase(),
      latitude: finalLocation.latitude,
      longitude: finalLocation.longitude,
      nickname: nickname || undefined,
      isGuest: true,
      guestFingerprint: getDeviceFingerprint(),
    };
    
    console.log('Attempting to join session...');
    joinSession.mutate(payload);
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

  const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      // Check file size (max 250MB)
      if (file.size > 250 * 1024 * 1024) {
        alert('File size must be less than 250MB');
        return;
      }
      
      if (!session || !participantId) {
        console.error('Cannot upload: no session or participant ID');
        alert('Please join the session first');
        return;
      }
      
      console.log('Uploading file:', file.name, formatFileSize(file.size));
      
      try {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('sessionId', session.id);
        formData.append('participantId', participantId);
        
        const response = await fetch('/api/upload', {
          method: 'POST',
          body: formData,
        });
        
        if (!response.ok) {
          const error = await response.json();
          throw new Error(error.error || 'Upload failed');
        }
        
        const result = await response.json();
        console.log('File uploaded successfully:', result);
        
        // Refresh session data to show new file
        refetch();
        
        // Reset the input
        event.target.value = '';
      } catch (error) {
        console.error('Upload error:', error);
        alert(`Failed to upload file: ${error instanceof Error ? error.message : 'Unknown error'}`);
        event.target.value = '';
      }
    }
  };

  // Check if session has expired
  const isExpired = session ? new Date(session.expiresAt) < new Date() : false;

  // Redirect to create new session if session not found
  useEffect(() => {
    if (!session && !sessionLoading) {
      router.push('/share');
    }
  }, [session, sessionLoading, router]);
  
  // Redirect to create new session if expired
  useEffect(() => {
    if (isExpired) {
      router.push('/share');
    }
  }, [isExpired, router]);

  if (!session) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-12">
            <Share2 className="w-16 h-16 text-primary mx-auto mb-4 animate-pulse" />
            <h3 className="text-xl font-semibold mb-2">Redirecting...</h3>
            <p className="text-default-500 mb-4">
              Setting up a share session for you
            </p>
            <Spinner size="lg" color="primary" />
          </CardBody>
        </Card>
      </div>
    );
  }

  if (isExpired) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-12">
            <Share2 className="w-16 h-16 text-primary mx-auto mb-4 animate-pulse" />
            <h3 className="text-xl font-semibold mb-2">Creating New Session...</h3>
            <p className="text-default-500 mb-4">
              The previous session has expired
            </p>
            <Spinner size="lg" color="primary" />
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
                  {session.participants.length}
                </Chip>
                <Chip
                  startContent={<Clock className="w-3 h-3" />}
                  size="sm"
                  variant="flat"
                  color={hoursRemaining < 1 ? 'warning' : 'default'}
                >
                  {hoursRemaining}h {minutesRemaining}m remaining
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
                onPress={() => {
                  // Trigger the hidden file input
                  const fileInput = document.getElementById('file-upload-input');
                  fileInput?.click();
                }}
              >
                Upload Files
              </Button>
              <input
                id="file-upload-input"
                type="file"
                className="hidden"
                onChange={handleFileSelect}
                accept="image/*,video/*,audio/*,.pdf,.doc,.docx,.txt,.md,.markdown,.json,.xml,.csv,.zip,.tar,.gz,.js,.ts,.jsx,.tsx,.css,.scss,.html,.py,.java,.cpp,.c,.h,.hpp,.rs,.go,.rb,.php,.swift,.kt,.dart,.yaml,.yml,.toml,.ini,.conf,.sh,.bash,.zsh,.fish,.ps1,.bat,.cmd"
              />
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