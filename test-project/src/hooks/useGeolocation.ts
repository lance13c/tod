'use client';

import { useState, useEffect, useCallback } from 'react';

export interface GeolocationState {
  latitude: number | null;
  longitude: number | null;
  accuracy: number | null;
  error: string | null;
  loading: boolean;
  permission: PermissionState | null;
}

export interface UseGeolocationOptions {
  enableHighAccuracy?: boolean;
  timeout?: number;
  maximumAge?: number;
  watchPosition?: boolean;
}

export function useGeolocation(options: UseGeolocationOptions = {}) {
  const [state, setState] = useState<GeolocationState>({
    latitude: null,
    longitude: null,
    accuracy: null,
    error: null,
    loading: false,
    permission: null,
  });

  const [watchId, setWatchId] = useState<number | null>(null);
  const [isSupported, setIsSupported] = useState<boolean | null>(null);

  // Check permission status
  const checkPermission = useCallback(async () => {
    if (!navigator.permissions) {
      return;
    }

    try {
      const result = await navigator.permissions.query({ name: 'geolocation' });
      setState(prev => ({ ...prev, permission: result.state }));
      
      result.addEventListener('change', () => {
        setState(prev => ({ ...prev, permission: result.state }));
      });
    } catch (error) {
      console.warn('Could not query geolocation permission:', error);
    }
  }, []);

  // Success callback
  const handleSuccess = useCallback((position: GeolocationPosition) => {
    setState({
      latitude: position.coords.latitude,
      longitude: position.coords.longitude,
      accuracy: position.coords.accuracy,
      error: null,
      loading: false,
      permission: 'granted',
    });
  }, []);

  // Error callback with retry logic
  const handleError = useCallback((error: GeolocationPositionError) => {
    let errorMessage = 'Unknown error occurred';
    
    switch (error.code) {
      case error.PERMISSION_DENIED:
        errorMessage = 'Location permission denied. Please enable location access.';
        setState(prev => ({ ...prev, permission: 'denied' }));
        break;
      case error.POSITION_UNAVAILABLE:
        errorMessage = 'Location information unavailable. Please try again.';
        // Try once more with lower accuracy
        if (navigator.geolocation) {
          navigator.geolocation.getCurrentPosition(
            handleSuccess,
            () => {
              // Final failure
              setState(prev => ({
                ...prev,
                error: errorMessage,
                loading: false,
              }));
            },
            {
              enableHighAccuracy: false, // Try with lower accuracy
              timeout: 10000,
              maximumAge: 60000, // Accept older cached position
            }
          );
          return;
        }
        break;
      case error.TIMEOUT:
        errorMessage = 'Location request timed out. Please try again.';
        break;
    }

    setState(prev => ({
      ...prev,
      error: errorMessage,
      loading: false,
    }));
  }, [handleSuccess]);

  // Get current position
  const getCurrentPosition = useCallback(() => {
    if (!navigator.geolocation) {
      setState(prev => ({
        ...prev,
        error: 'Geolocation is not supported by your browser',
        loading: false,
      }));
      return;
    }

    setState(prev => ({ ...prev, loading: true, error: null }));

    const geoOptions: PositionOptions = {
      enableHighAccuracy: options.enableHighAccuracy ?? true,
      timeout: options.timeout ?? 30000, // Increased timeout to 30 seconds
      maximumAge: options.maximumAge ?? 5000, // Allow cached position up to 5 seconds old
    };

    if (options.watchPosition) {
      const id = navigator.geolocation.watchPosition(
        handleSuccess,
        handleError,
        geoOptions
      );
      setWatchId(id);
    } else {
      navigator.geolocation.getCurrentPosition(
        handleSuccess,
        handleError,
        geoOptions
      );
    }
  }, [options, handleSuccess, handleError]);

  // Stop watching position
  const stopWatching = useCallback(() => {
    if (watchId !== null) {
      navigator.geolocation.clearWatch(watchId);
      setWatchId(null);
    }
  }, [watchId]);

  // Request permission and get location
  const requestLocation = useCallback(async () => {
    await checkPermission();
    getCurrentPosition();
  }, [checkPermission, getCurrentPosition]);

  // Check permission and browser support on mount
  useEffect(() => {
    setIsSupported(typeof navigator !== 'undefined' && 'geolocation' in navigator);
    checkPermission();
  }, [checkPermission]);

  // Cleanup
  useEffect(() => {
    return () => {
      stopWatching();
    };
  }, [stopWatching]);

  return {
    ...state,
    requestLocation,
    getCurrentPosition,
    stopWatching,
    isSupported,
  };
}