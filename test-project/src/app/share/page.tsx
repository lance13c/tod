'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Card, CardBody, CardHeader, Button, Input, Textarea, Slider, Chip, Spinner, Alert } from '@nextui-org/react';
import { LocationPermission } from '@/components/location/LocationPermission';
import { Users, Clock, MapPin, Zap, Shield, Share2, Building, AlertCircle, ArrowRight } from 'lucide-react';
import { api } from '@/lib/trpc/client';
import { useGeolocation } from '@/hooks/useGeolocation';

// Generate a device fingerprint for guest users
function getDeviceFingerprint(): string {
  if (typeof window === 'undefined') return '';
  
  const stored = localStorage.getItem('device_fingerprint');
  if (stored) return stored;
  
  const fingerprint = Math.random().toString(36).substring(2) + Date.now().toString(36);
  localStorage.setItem('device_fingerprint', fingerprint);
  return fingerprint;
}

export default function SharePage() {
  const router = useRouter();
  const [step, setStep] = useState<'detecting' | 'no-building' | 'existing-session' | 'create'>('detecting');
  const [sessionName, setSessionName] = useState('');
  const [sessionDescription, setSessionDescription] = useState('');
  const [radius, setRadius] = useState(100);
  const [expiresIn, setExpiresIn] = useState(4);
  const [maxParticipants, setMaxParticipants] = useState(10);
  const [buildingInfo, setBuildingInfo] = useState<{ id: string; name: string } | null>(null);
  const [existingSession, setExistingSession] = useState<any>(null);

  const {
    latitude,
    longitude,
    error: locationError,
    loading: locationLoading,
    requestLocation,
    permission,
  } = useGeolocation({ enableHighAccuracy: true });

  // Query for building at location
  const { data: buildingData, isLoading: buildingLoading } = api.shareSession.findBuildingAtLocation.useQuery(
    { 
      latitude: latitude!, 
      longitude: longitude! 
    },
    { 
      enabled: !!latitude && !!longitude,
    }
  );

  // Handle building data response
  useEffect(() => {
    if (buildingData) {
      if (buildingData.building) {
        setBuildingInfo(buildingData.building);
        if (buildingData.existingSession) {
          setExistingSession(buildingData.existingSession);
          setStep('existing-session');
        } else {
          // Auto-fill session name with building name
          setSessionName(`${buildingData.building.name} - File Share`);
          setStep('create');
        }
      } else {
        setStep('no-building');
      }
    }
  }, [buildingData]);

  const createSession = api.shareSession.createSession.useMutation();

  // Handle successful session creation
  useEffect(() => {
    if (createSession.isSuccess && createSession.data) {
      router.push(`/share/${createSession.data.code}`);
    }
  }, [createSession.isSuccess, createSession.data, router]);

  const handleCreateSession = async () => {
    if (!latitude || !longitude || !sessionName) return;

    createSession.mutate({
      name: sessionName,
      description: sessionDescription || undefined,
      latitude,
      longitude,
      geoLockRadius: radius,
      maxParticipants,
      expiresInHours: expiresIn,
      requiresAuth: false,
      isGuest: true,
      guestFingerprint: getDeviceFingerprint(),
    });
  };

  const handleJoinExisting = () => {
    if (existingSession) {
      router.push(`/share/${existingSession.code}`);
    }
  };

  // Auto-request location on mount
  useEffect(() => {
    if (!latitude && !longitude && !locationLoading && permission !== 'denied') {
      requestLocation();
    }
  }, []);

  // Show location permission screen if needed
  if (!latitude || !longitude) {
    if (permission === 'denied' || locationError) {
      return (
        <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
          <Card className="max-w-md w-full">
            <CardBody className="text-center py-8">
              <AlertCircle className="w-16 h-16 text-danger mx-auto mb-4" />
              <h3 className="text-xl font-semibold mb-2">Location Required</h3>
              <p className="text-default-500 mb-4">
                {locationError || 'Location access was denied. GroupUp needs your location to connect you with others in your building.'}
              </p>
              <p className="text-sm text-default-400 mb-4">
                Please enable location access in your browser settings and refresh the page.
              </p>
              <Button color="primary" onPress={() => window.location.reload()}>
                Try Again
              </Button>
            </CardBody>
          </Card>
        </div>
      );
    }

    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <LocationPermission
          onLocationGranted={() => {}}
          title="Share Files in Your Building"
          description="GroupUp will detect your building and connect you with others nearby"
        />
      </div>
    );
  }

  // Loading state while detecting building
  if (step === 'detecting' || buildingLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-12">
            <Building className="w-16 h-16 text-primary mx-auto mb-4 animate-pulse" />
            <h3 className="text-xl font-semibold mb-2">Detecting Your Building...</h3>
            <p className="text-default-500 mb-4">
              Checking if you're inside a building
            </p>
            <Spinner size="lg" color="primary" />
          </CardBody>
        </Card>
      </div>
    );
  }

  // No building found
  if (step === 'no-building') {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-8">
            <MapPin className="w-16 h-16 text-warning mx-auto mb-4" />
            <h3 className="text-xl font-semibold mb-2">No Building Detected</h3>
            <p className="text-default-500 mb-6">
              You don't appear to be inside a building. You can still create a location-based session.
            </p>
            <div className="flex gap-3 justify-center">
              <Button 
                variant="flat" 
                onPress={() => router.push('/')}
              >
                Go Back
              </Button>
              <Button 
                color="primary" 
                onPress={() => {
                  setSessionName('Outdoor File Share');
                  setStep('create');
                }}
              >
                Create Session Anyway
              </Button>
            </div>
          </CardBody>
        </Card>
      </div>
    );
  }

  // Existing session found
  if (step === 'existing-session' && existingSession) {
    const timeRemaining = new Date(existingSession.expiresAt).getTime() - Date.now();
    const hoursRemaining = Math.floor(timeRemaining / (1000 * 60 * 60));
    const minutesRemaining = Math.floor((timeRemaining % (1000 * 60 * 60)) / (1000 * 60));

    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="py-8">
            <div className="text-center mb-6">
              <Building className="w-16 h-16 text-success mx-auto mb-4" />
              <h3 className="text-xl font-semibold mb-2">Active Session Found!</h3>
              <p className="text-default-500">
                There's already a sharing session in {buildingInfo?.name}
              </p>
            </div>

            <Alert 
              color="primary" 
              variant="flat"
              className="mb-6"
              title={existingSession.name}
              description={
                <div className="mt-2 space-y-1 text-sm">
                  <div className="flex items-center gap-2">
                    <Users className="w-4 h-4" />
                    <span>{existingSession.participantCount} participants</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Clock className="w-4 h-4" />
                    <span>{hoursRemaining}h {minutesRemaining}m remaining</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Share2 className="w-4 h-4" />
                    <span className="font-mono font-bold">{existingSession.code}</span>
                  </div>
                </div>
              }
            />

            <div className="flex gap-3">
              <Button 
                variant="flat" 
                onPress={() => {
                  setStep('create');
                }}
                className="flex-1"
              >
                Create New Session
              </Button>
              <Button 
                color="primary" 
                onPress={handleJoinExisting}
                endContent={<ArrowRight className="w-4 h-4" />}
                className="flex-1"
              >
                Join Existing
              </Button>
            </div>
          </CardBody>
        </Card>
      </div>
    );
  }

  // Create session form
  return (
    <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4">
      <div className="max-w-2xl mx-auto pt-10">
        <Card>
          <CardHeader className="flex flex-col gap-1 pb-6">
            <h2 className="text-2xl font-bold">Create Sharing Session</h2>
            {buildingInfo && (
              <div className="flex items-center gap-2 text-default-500">
                <Building className="w-4 h-4" />
                <span>{buildingInfo.name}</span>
              </div>
            )}
          </CardHeader>
          <CardBody className="gap-6">
            <Input
              label="Session Name"
              placeholder="e.g., Team Meeting Files"
              value={sessionName}
              onValueChange={setSessionName}
              isRequired
              description="Give your session a friendly name"
            />

            <Textarea
              label="Description (Optional)"
              placeholder="What files will be shared?"
              value={sessionDescription}
              onValueChange={setSessionDescription}
              maxRows={3}
            />

            <div className="space-y-4">
              <div>
                <div className="flex justify-between items-center mb-2">
                  <label className="text-sm font-medium">
                    Location Radius
                  </label>
                  <Chip size="sm" variant="flat">
                    {radius}m
                  </Chip>
                </div>
                <Slider
                  size="sm"
                  step={10}
                  minValue={10}
                  maxValue={500}
                  value={radius}
                  onChange={(value) => setRadius(value as number)}
                  className="mb-1"
                  startContent={<MapPin className="w-4 h-4 text-default-400" />}
                />
                <p className="text-xs text-default-400">
                  Only people within this distance can join
                </p>
              </div>

              <div>
                <div className="flex justify-between items-center mb-2">
                  <label className="text-sm font-medium">
                    Session Duration
                  </label>
                  <Chip size="sm" variant="flat">
                    {expiresIn} hours
                  </Chip>
                </div>
                <Slider
                  size="sm"
                  step={1}
                  minValue={1}
                  maxValue={24}
                  value={expiresIn}
                  onChange={(value) => setExpiresIn(value as number)}
                  className="mb-1"
                  startContent={<Clock className="w-4 h-4 text-default-400" />}
                />
                <p className="text-xs text-default-400">
                  Session will automatically end after this time
                </p>
              </div>

              <div>
                <div className="flex justify-between items-center mb-2">
                  <label className="text-sm font-medium">
                    Max Participants
                  </label>
                  <Chip size="sm" variant="flat">
                    {maxParticipants} people
                  </Chip>
                </div>
                <Slider
                  size="sm"
                  step={1}
                  minValue={2}
                  maxValue={50}
                  value={maxParticipants}
                  onChange={(value) => setMaxParticipants(value as number)}
                  className="mb-1"
                  startContent={<Users className="w-4 h-4 text-default-400" />}
                />
                <p className="text-xs text-default-400">
                  Maximum number of people who can join
                </p>
              </div>
            </div>

            <div className="flex gap-3">
              <Button
                variant="flat"
                onPress={() => router.push('/')}
              >
                Cancel
              </Button>
              <Button
                color="primary"
                onPress={handleCreateSession}
                isLoading={createSession.isPending}
                isDisabled={!sessionName}
                className="flex-1"
              >
                Create Session
              </Button>
            </div>
          </CardBody>
        </Card>
      </div>
    </div>
  );
}