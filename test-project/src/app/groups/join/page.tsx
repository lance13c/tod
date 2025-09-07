"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "@/lib/auth-client";
import {
  Card,
  CardBody,
  CardHeader,
  Button,
  Chip,
  Spinner,
  Avatar,
  Badge,
  Divider,
} from "@nextui-org/react";
import {
  MapPin,
  Users,
  ArrowRight,
  AlertCircle,
  CheckCircle,
  ArrowLeft,
  Clock,
  Building2,
  RefreshCw,
} from "lucide-react";
import dynamic from "next/dynamic";

// Dynamic import for the BuildingMap component to avoid SSR issues
const BuildingMap = dynamic(() => import("@/components/BuildingMap"), {
  ssr: false,
  loading: () => (
    <div className="flex justify-center items-center h-96 bg-gray-100 dark:bg-gray-800 rounded-lg">
      <Spinner size="lg" />
    </div>
  ),
});

interface LocationData {
  latitude: number;
  longitude: number;
  accuracy: number;
  address?: string;
  nearestBuilding?: {
    distance: number;
    isInside: boolean;
    centroid: [number, number];
    geometry?: any;
    name?: string;
    address?: string;
  };
}

interface NearbyGroup {
  id: string;
  name: string;
  description?: string;
  distance: number;
  canJoin: boolean;
  isMember: boolean;
  latitude: number;
  longitude: number;
  radius: number;
  expiresAt: string;
  organization?: {
    name: string;
    logoUrl?: string;
    brandColor?: string;
  };
  members: any[];
  _count: {
    files: number;
  };
}

export default function JoinGroupPage() {
  const router = useRouter();
  const { data: session } = useSession();
  const [location, setLocation] = useState<LocationData | null>(null);
  const [gettingLocation, setGettingLocation] = useState(false);
  const [nearbyGroups, setNearbyGroups] = useState<NearbyGroup[]>([]);
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [joining, setJoining] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Automatically get location when page loads
    getCurrentLocation();
  }, []);

  useEffect(() => {
    // When location is available, fetch nearby groups
    if (location) {
      fetchNearbyGroups();
    }
  }, [location]);

  const getCurrentLocation = async () => {
    setGettingLocation(true);
    setError(null);

    if (!navigator.geolocation) {
      setError("Geolocation is not supported by your browser");
      setGettingLocation(false);
      return;
    }

    navigator.geolocation.getCurrentPosition(
      async (position) => {
        const locationData: LocationData = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracy: position.coords.accuracy,
        };

        // Try to get address from coordinates
        try {
          const response = await fetch(
            `https://nominatim.openstreetmap.org/reverse?format=json&lat=${locationData.latitude}&lon=${locationData.longitude}`
          );
          const data = await response.json();
          locationData.address = data.display_name;
        } catch (err) {
          console.error("Failed to get address:", err);
        }

        // Find nearest building using API call
        try {
          const response = await fetch('/api/buildings/nearest', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              latitude: locationData.latitude,
              longitude: locationData.longitude,
              bufferMeters: 100
            })
          });
          
          if (response.ok) {
            const building = await response.json();
            if (building) {
              locationData.nearestBuilding = {
                distance: building.distance,
                isInside: building.isInside,
                centroid: building.centroid,
                geometry: building.geometry,
                name: building.name,
                address: building.address,
              };
            }
          }
        } catch (err) {
          console.error("Failed to find nearest building:", err);
        }

        setLocation(locationData);
        setGettingLocation(false);
      },
      (error) => {
        setError(`Failed to get location: ${error.message}`);
        setGettingLocation(false);
      },
      {
        enableHighAccuracy: true,
        timeout: 30000,
        maximumAge: 60000,
      }
    );
  };

  const fetchNearbyGroups = async () => {
    if (!location) return;

    setLoadingGroups(true);
    try {
      const response = await fetch("/api/groups/nearby", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          latitude: location.latitude,
          longitude: location.longitude,
          maxDistance: 500, // Show groups within 500m
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to fetch nearby groups");
      }

      const groups = await response.json();
      setNearbyGroups(groups);
    } catch (err: any) {
      setError(err.message || "Failed to fetch nearby groups");
    } finally {
      setLoadingGroups(false);
    }
  };

  const handleJoinGroup = async (groupId: string) => {
    if (!location) {
      setError("Location is required to join a group");
      return;
    }

    setJoining(groupId);
    setError(null);

    try {
      const response = await fetch(`/api/groups/${groupId}/join`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          latitude: location.latitude,
          longitude: location.longitude,
        }),
      });
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to join group");
      }
      
      // Redirect to the group page
      router.push(`/groups/${groupId}`);
    } catch (err: any) {
      setError(err.message || "Failed to join group");
      setJoining(null);
    }
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

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      {/* Header */}
      <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-md shadow-sm border-b dark:border-gray-700">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center gap-3">
              <Button
                variant="light"
                isIconOnly
                onPress={() => router.push("/dashboard")}
              >
                <ArrowLeft className="w-5 h-5" />
              </Button>
              <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                  Join Nearby Group
                </h1>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Find and join groups in your location
                </p>
              </div>
            </div>
            <Button
              variant="light"
              size="sm"
              onPress={() => {
                getCurrentLocation();
                if (location) fetchNearbyGroups();
              }}
              startContent={<RefreshCw className="w-4 h-4" />}
            >
              Refresh
            </Button>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Location Status */}
        <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90 shadow-xl mb-6">
          <CardBody>
            <div className="flex items-start gap-3">
              {gettingLocation ? (
                <>
                  <Spinner size="sm" color="primary" />
                  <div>
                    <p className="font-medium text-sm">Getting your location...</p>
                    <p className="text-xs text-gray-500 mt-1">
                      Please allow location access when prompted
                    </p>
                  </div>
                </>
              ) : location ? (
                <>
                  <CheckCircle className="w-5 h-5 text-green-500 mt-0.5" />
                  <div className="flex-1">
                    <p className="font-medium text-sm text-green-700 dark:text-green-400">
                      Location detected
                    </p>
                    {location.nearestBuilding?.isInside ? (
                      <p className="text-xs text-blue-600 dark:text-blue-400 font-medium flex items-center gap-1 mt-1">
                        <Building2 className="w-3 h-3" />
                        Inside building
                      </p>
                    ) : location.nearestBuilding ? (
                      <p className="text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1 mt-1">
                        <Building2 className="w-3 h-3" />
                        {Math.round(location.nearestBuilding.distance)}m from nearest building
                      </p>
                    ) : null}
                    <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                      {location.address || `${location.latitude.toFixed(4)}, ${location.longitude.toFixed(4)}`}
                    </p>
                  </div>
                  <Button
                    size="sm"
                    variant="light"
                    onPress={getCurrentLocation}
                    isIconOnly
                  >
                    <MapPin className="w-4 h-4" />
                  </Button>
                </>
              ) : (
                <>
                  <AlertCircle className="w-5 h-5 text-amber-500 mt-0.5" />
                  <div className="flex-1">
                    <p className="font-medium text-sm">Location required</p>
                    <p className="text-xs text-gray-500 mt-1">
                      We need your location to find nearby groups
                    </p>
                  </div>
                  <Button
                    size="sm"
                    color="primary"
                    variant="flat"
                    onPress={getCurrentLocation}
                    startContent={<MapPin className="w-4 h-4" />}
                  >
                    Enable
                  </Button>
                </>
              )}
            </div>
          </CardBody>
        </Card>

        {/* Building Map - show when we have a building detected */}
        {location?.nearestBuilding && (
          <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90 shadow-xl mb-6">
            <CardHeader>
              <div className="flex items-center gap-2">
                <Building2 className="w-5 h-5 text-blue-500" />
                <div className="flex-1">
                  <h3 className="font-semibold text-sm">
                    {location.nearestBuilding.isInside ? "You're Inside" : "Nearest Building"}
                  </h3>
                  {location.nearestBuilding.name && (
                    <p className="text-xs text-gray-500">{location.nearestBuilding.name}</p>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardBody className="p-0">
              <BuildingMap
                buildingGeometry={location.nearestBuilding.geometry}
                buildingCentroid={location.nearestBuilding.centroid}
                groupLocation={[location.latitude, location.longitude]}
                radius={50}
                height="300px"
              />
            </CardBody>
          </Card>
        )}

        {/* Nearby Groups */}
        {loadingGroups ? (
          <div className="flex justify-center py-12">
            <Spinner size="lg" />
          </div>
        ) : nearbyGroups.length > 0 ? (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
              Nearby Groups ({nearbyGroups.length})
            </h2>
            {nearbyGroups.map((group) => (
              <Card 
                key={group.id}
                className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90"
                style={{
                  borderLeft: group.organization?.brandColor 
                    ? `4px solid ${group.organization.brandColor}`
                    : undefined,
                }}
              >
                <CardBody>
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                      {group.organization && (
                        <Avatar
                          src={group.organization.logoUrl}
                          size="md"
                        />
                      )}
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-gray-900 dark:text-white">
                            {group.name}
                          </h3>
                          <Badge
                            color={group.distance <= 50 ? "success" : group.distance <= 100 ? "warning" : "default"}
                            variant="flat"
                            size="sm"
                          >
                            {group.distance}m away
                          </Badge>
                        </div>
                        {group.organization && (
                          <p className="text-xs text-gray-500 dark:text-gray-400">
                            {group.organization.name}
                          </p>
                        )}
                        {group.description && (
                          <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                            {group.description}
                          </p>
                        )}
                        <div className="flex items-center gap-4 mt-3 text-sm text-gray-500">
                          <span className="flex items-center gap-1">
                            <Users className="w-4 h-4" />
                            {group.members.length} members
                          </span>
                          <span className="flex items-center gap-1">
                            <Clock className="w-4 h-4" />
                            {getTimeRemaining(group.expiresAt)}
                          </span>
                          <span className="flex items-center gap-1">
                            <MapPin className="w-4 h-4" />
                            {group.radius}m radius
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-2">
                      {group.isMember ? (
                        <Button
                          size="sm"
                          color="default"
                          variant="solid"
                          onPress={() => router.push(`/groups/${group.id}`)}
                        >
                          View Group
                        </Button>
                      ) : group.canJoin ? (
                        <Button
                          size="sm"
                          color="primary"
                          isLoading={joining === group.id}
                          onPress={() => handleJoinGroup(group.id)}
                          startContent={<Users className="w-4 h-4" />}
                        >
                          Join Group
                        </Button>
                      ) : (
                        <Chip color="danger" variant="flat" size="sm">
                          Too far away
                        </Chip>
                      )}
                    </div>
                  </div>
                </CardBody>
              </Card>
            ))}
          </div>
        ) : location && !loadingGroups ? (
          <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90">
            <CardBody className="text-center py-12">
              <MapPin className="w-12 h-12 text-gray-300 mx-auto mb-3" />
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                No Groups Nearby
              </h3>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                There are no active groups within 500m of your location.
              </p>
              <Button
                color="primary"
                variant="flat"
                className="mt-4"
                onPress={() => router.push("/groups/new")}
                startContent={<Users className="w-4 h-4" />}
              >
                Create a Group
              </Button>
            </CardBody>
          </Card>
        ) : null}

        {/* Error Display */}
        {error && (
          <Chip color="danger" variant="flat" className="w-full">
            <span className="text-sm">{error}</span>
          </Chip>
        )}
      </div>
    </div>
  );
}