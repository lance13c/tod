'use client';

import { Button, Card, CardBody, CardHeader } from '@nextui-org/react';
import { MapPin, Shield, AlertCircle } from 'lucide-react';
import { useGeolocation } from '@/hooks/useGeolocation';

interface LocationPermissionProps {
  onLocationGranted: (latitude: number, longitude: number) => void;
  title?: string;
  description?: string;
}

export function LocationPermission({ 
  onLocationGranted,
  title = "Enable Location Access",
  description = "We need your location to connect you with others in your building"
}: LocationPermissionProps) {
  const {
    latitude,
    longitude,
    error,
    loading,
    permission,
    requestLocation,
    isSupported
  } = useGeolocation();

  // Call callback when location is obtained
  if (latitude && longitude) {
    onLocationGranted(latitude, longitude);
  }

  if (!isSupported) {
    return (
      <Card className="max-w-md mx-auto">
        <CardBody className="text-center py-8">
          <AlertCircle className="w-16 h-16 text-warning mx-auto mb-4" />
          <h3 className="text-xl font-semibold mb-2">Location Not Supported</h3>
          <p className="text-default-500">
            Your browser doesn't support location services. 
            Please try a different browser or device.
          </p>
        </CardBody>
      </Card>
    );
  }

  if (permission === 'denied') {
    return (
      <Card className="max-w-md mx-auto">
        <CardBody className="text-center py-8">
          <AlertCircle className="w-16 h-16 text-danger mx-auto mb-4" />
          <h3 className="text-xl font-semibold mb-2">Location Access Blocked</h3>
          <p className="text-default-500 mb-4">
            You've blocked location access. To continue, please:
          </p>
          <ol className="text-left text-sm text-default-500 space-y-2 mb-4">
            <li>1. Click the lock icon in your browser's address bar</li>
            <li>2. Find "Location" in the permissions</li>
            <li>3. Change it to "Allow"</li>
            <li>4. Refresh this page</li>
          </ol>
        </CardBody>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="max-w-md mx-auto">
        <CardBody className="text-center py-8">
          <AlertCircle className="w-16 h-16 text-warning mx-auto mb-4" />
          <h3 className="text-xl font-semibold mb-2">Location Error</h3>
          <p className="text-default-500 mb-4">{error}</p>
          <Button 
            color="primary" 
            onPress={requestLocation}
            isLoading={loading}
          >
            Try Again
          </Button>
        </CardBody>
      </Card>
    );
  }

  if (permission === 'granted' && loading) {
    return (
      <Card className="max-w-md mx-auto">
        <CardBody className="text-center py-8">
          <div className="animate-pulse">
            <MapPin className="w-16 h-16 text-primary mx-auto mb-4" />
          </div>
          <h3 className="text-xl font-semibold mb-2">Getting Your Location...</h3>
          <p className="text-default-500">
            Please wait while we determine your location
          </p>
        </CardBody>
      </Card>
    );
  }

  return (
    <Card className="max-w-md mx-auto">
      <CardHeader className="flex flex-col items-center pb-0 pt-8">
        <div className="relative">
          <MapPin className="w-20 h-20 text-primary" />
          <div className="absolute -bottom-1 -right-1 bg-success rounded-full p-1">
            <Shield className="w-4 h-4 text-white" />
          </div>
        </div>
      </CardHeader>
      <CardBody className="text-center py-6">
        <h2 className="text-2xl font-bold mb-3">{title}</h2>
        <p className="text-default-500 mb-6">
          {description}
        </p>
        
        <div className="bg-default-100 rounded-lg p-4 mb-6 text-left">
          <h4 className="font-semibold mb-2 flex items-center gap-2">
            <Shield className="w-4 h-4 text-success" />
            Your Privacy Matters
          </h4>
          <ul className="text-sm text-default-600 space-y-1">
            <li>• Location is only used to find your building</li>
            <li>• We don't store your exact coordinates</li>
            <li>• Location data stays on your device</li>
            <li>• You can disable access anytime</li>
          </ul>
        </div>

        <Button 
          color="primary" 
          size="lg"
          onPress={requestLocation}
          isLoading={loading}
          startContent={<MapPin className="w-5 h-5" />}
          className="font-semibold"
        >
          Enable Location Access
        </Button>
        
        <p className="text-xs text-default-400 mt-4">
          By enabling location, you agree to our privacy policy
        </p>
      </CardBody>
    </Card>
  );
}