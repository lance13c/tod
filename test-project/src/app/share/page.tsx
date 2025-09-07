'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Card, CardBody, Button, Spinner, Alert } from '@nextui-org/react';
import { LocationPermission } from '@/components/location/LocationPermission';
import { Users, Clock, Share2, Building, AlertCircle, ArrowRight } from 'lucide-react';
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
  const [step, setStep] = useState<'detecting' | 'no-building' | 'existing-session' | 'creating'>('detecting');
  const [sessionName, setSessionName] = useState('');
  const radius = 500; // Fixed radius of 500m
  const maxParticipants = 50; // Fixed max participants
  const [buildingInfo, setBuildingInfo] = useState<{ id: string; name: string } | null>(null);
  const [existingSession, setExistingSession] = useState<any>(null);
  const [usingFallback, setUsingFallback] = useState(false);

  // Fallback coordinates
  const FALLBACK_LAT = 36.14879834580897;
  const FALLBACK_LONG = -86.80765485270973;

  const {
    latitude,
    longitude,
    error: locationError,
    loading: locationLoading,
    requestLocation,
    permission,
  } = useGeolocation({ enableHighAccuracy: false }); // Use lower accuracy for faster results

  // Determine which coordinates to use (actual or fallback)
  const effectiveLat = latitude || (usingFallback ? FALLBACK_LAT : null);
  const effectiveLong = longitude || (usingFallback ? FALLBACK_LONG : null);

  // Query for building at location
  const { data: buildingData, isLoading: buildingLoading } = api.shareSession.findBuildingAtLocation.useQuery(
    { 
      latitude: effectiveLat!, 
      longitude: effectiveLong! 
    },
    { 
      enabled: !!effectiveLat && !!effectiveLong,
    }
  );

  // Handle building data response and auto-create session
  useEffect(() => {
    if (buildingData) {
      if (buildingData.building) {
        setBuildingInfo(buildingData.building);
        if (buildingData.existingSession) {
          setExistingSession(buildingData.existingSession);
          setStep('existing-session');
        } else {
          // Auto-create session with building/street name
          const buildingName = buildingData.building.name || 'Building';
          setSessionName(`${buildingName} Share`);
          setStep('creating'); // New state for auto-creation
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

  // Handle no building case - transition to creating
  useEffect(() => {
    if (step === 'no-building' && !sessionName) {
      setSessionName('Outdoor Share');
      setStep('creating');
    }
  }, [step, sessionName]);

  // Auto-create session when step changes to 'creating'
  useEffect(() => {
    if (step === 'creating' && sessionName && effectiveLat && effectiveLong && !createSession.isPending && !createSession.isSuccess) {
      createSession.mutate({
        name: sessionName,
        description: '', // No description needed for auto-creation
        latitude: effectiveLat,
        longitude: effectiveLong,
        geoLockRadius: radius,
        maxParticipants,
        expiresInHours: 4, // Default 4 hours
        requiresAuth: false,
        isGuest: true,
        guestFingerprint: getDeviceFingerprint(),
      });
    }
  }, [step, sessionName, effectiveLat, effectiveLong]); // Removed createSession.isPending from deps to prevent re-triggers

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
  if (!effectiveLat || !effectiveLong) {
    if (permission === 'denied' || locationError) {
      // Use fallback location on error
      if (!usingFallback) {
        setUsingFallback(true);
        return (
          <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
            <Card className="max-w-md w-full">
              <CardBody className="text-center py-8">
                <AlertCircle className="w-16 h-16 text-warning mx-auto mb-4" />
                <h3 className="text-xl font-semibold mb-2">Using Default Location</h3>
                <p className="text-default-500 mb-4">
                  Location access failed. Using a default location for testing.
                </p>
                <Spinner size="lg" color="primary" />
              </CardBody>
            </Card>
          </div>
        );
      }
    }

    if (!usingFallback) {
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

  // Creating session state
  if (step === 'creating' || createSession.isPending) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-12">
            <Share2 className="w-16 h-16 text-primary mx-auto mb-4 animate-pulse" />
            <h3 className="text-xl font-semibold mb-2">Creating Share Session...</h3>
            <p className="text-default-500 mb-4">
              {sessionName}
            </p>
            <Spinner size="lg" color="primary" />
          </CardBody>
        </Card>
      </div>
    );
  }

  // No building found - show creating state
  if (step === 'no-building') {
    return (
      <div className="min-h-screen bg-gradient-to-br from-primary-50 to-secondary-50 dark:from-gray-900 dark:to-gray-800 p-4 flex items-center justify-center">
        <Card className="max-w-md w-full">
          <CardBody className="text-center py-12">
            <Share2 className="w-16 h-16 text-primary mx-auto mb-4 animate-pulse" />
            <h3 className="text-xl font-semibold mb-2">Creating Outdoor Session...</h3>
            <p className="text-default-500 mb-4">
              Setting up a location-based share
            </p>
            <Spinner size="lg" color="primary" />
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

  // Fallback - should not reach here
  return null;
}