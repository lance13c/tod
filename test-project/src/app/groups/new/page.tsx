"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "@/lib/auth-client";
import {
  Card,
  CardBody,
  CardHeader,
  Input,
  Button,
  Textarea,
  Chip,
  Select,
  SelectItem,
  Spinner,
  Avatar,
} from "@nextui-org/react";
import {
  MapPin,
  Users,
  Clock,
  ArrowRight,
  Sparkles,
  Building2,
  AlertCircle,
  CheckCircle,
  ArrowLeft,
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

interface Organization {
  id: string;
  name: string;
  logoUrl?: string;
  brandColor?: string;
}

export default function NewGroupPage() {
  const router = useRouter();
  const { data: session, isPending } = useSession();
  const [groupName, setGroupName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedOrgId, setSelectedOrgId] = useState("");
  const [radius, setRadius] = useState("100");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [location, setLocation] = useState<LocationData | null>(null);
  const [gettingLocation, setGettingLocation] = useState(false);
  const [organizations, setOrganizations] = useState<Organization[]>([]);

  useEffect(() => {
    if (!isPending && !session) {
      router.push("/login");
    }
  }, [session, isPending, router]);

  useEffect(() => {
    // Don't auto-request location on mount to avoid immediate errors
    // Let user click Enable button instead
    // getCurrentLocation();
    // Fetch user's organizations
    fetchOrganizations();
  }, []);

  // Debug logging for location changes
  useEffect(() => {
    console.log("🔄 Location state updated:", {
      hasLocation: !!location,
      hasBuilding: !!location?.nearestBuilding,
      buildingDetails: location?.nearestBuilding
    });
  }, [location]);

  const fetchOrganizations = async () => {
    // TODO: Replace with actual API call
    // const response = await fetch("/api/organizations/my");
    // const data = await response.json();
    // setOrganizations(data);
    
    // Mock data for now
    setOrganizations([
      {
        id: "org-1",
        name: "Acme Corporation",
        logoUrl: "https://api.dicebear.com/7.x/identicon/svg?seed=acme",
        brandColor: "#FF6B6B",
      },
      {
        id: "org-2",
        name: "TechStart Inc",
        logoUrl: "https://api.dicebear.com/7.x/identicon/svg?seed=techstart",
        brandColor: "#4A90E2",
      },
    ]);
  };

  const getCurrentLocation = async () => {
    setGettingLocation(true);
    setError(null);

    if (!navigator.geolocation) {
      setError("Geolocation is not supported by your browser");
      setGettingLocation(false);
      return;
    }

    // Check permissions if available
    try {
      const permissionStatus = await navigator.permissions.query({ name: 'geolocation' as PermissionName });
      
      if (permissionStatus.state === 'denied') {
        setError("Location permission denied. Please enable location access in your browser settings.");
        setGettingLocation(false);
        return;
      }
    } catch (err) {
      // Permissions API might not be available, continue with getCurrentPosition
      console.log("Permissions API not available, continuing with location request");
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
        console.log("🏢 Searching for building at:", {
          latitude: locationData.latitude,
          longitude: locationData.longitude
        });
        
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
            console.log("🏢 Building API Response:", building);
            
            if (building) {
              locationData.nearestBuilding = {
                distance: building.distance,
                isInside: building.isInside,
                centroid: building.centroid,
                geometry: building.geometry,
                name: building.name,
                address: building.address,
              };
              
              console.log("✅ Building found and set:", {
                name: building.name,
                isInside: building.isInside,
                distance: building.distance,
                centroid: building.centroid,
                hasGeometry: !!building.geometry
              });
            } else {
              console.log("ℹ️ No building found within range (this is okay)");
            }
          } else {
            console.warn("⚠️ Building API returned status:", response.status, "- continuing without building data");
          }
        } catch (err) {
          console.warn("⚠️ Could not fetch building data:", err, "- continuing without it");
        }

        setLocation(locationData);
        console.log("📍 Final location state set:", {
          hasLocation: !!locationData,
          hasBuilding: !!locationData.nearestBuilding,
          buildingName: locationData.nearestBuilding?.name
        });
        setGettingLocation(false);
      },
      (error) => {
        // Provide more user-friendly error messages
        let errorMessage = "Unable to get your location. ";
        
        switch(error.code) {
          case error.PERMISSION_DENIED:
            errorMessage = "Location permission denied. Please allow location access and try again.";
            break;
          case error.POSITION_UNAVAILABLE:
            errorMessage = "Location information is unavailable. Please check your device settings.";
            break;
          case error.TIMEOUT:
            errorMessage = "Location request timed out. Please try again.";
            break;
          default:
            errorMessage = "An error occurred while getting your location. Please try again.";
        }
        
        setError(errorMessage);
        setGettingLocation(false);
      },
      {
        enableHighAccuracy: true,
        timeout: 30000,
        maximumAge: 60000,
      }
    );
  };

  const handleCreateGroup = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!location) {
      setError("Please enable location access to create a group");
      return;
    }

    if (!groupName.trim()) {
      setError("Please enter a group name");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch("/api/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: groupName,
          description,
          latitude: location.latitude,
          longitude: location.longitude,
          radius: parseInt(radius),
          organizationId: selectedOrgId || null,
        }),
      });
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to create group");
      }
      
      const group = await response.json();
      router.push(`/groups/${group.id}`);
    } catch (err: any) {
      setError(err.message || "Failed to create group");
    } finally {
      setLoading(false);
    }
  };

  const selectedOrg = organizations.find((org) => org.id === selectedOrgId);

  if (isPending) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

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
                  Create New Group
                </h1>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Start a location-based file sharing session
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Form Section */}
          <div className="lg:col-span-2">
            <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90 shadow-xl">
              <CardBody>
                <form onSubmit={handleCreateGroup} className="space-y-6">
                  {/* Location Status */}
                  <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-gray-700 dark:to-gray-700 rounded-lg p-4">
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
                              <div className="mt-1">
                                <p className="text-xs text-blue-600 dark:text-blue-400 font-medium flex items-center gap-1">
                                  <Building2 className="w-3 h-3" />
                                  Inside building
                                </p>
                              </div>
                            ) : location.nearestBuilding ? (
                              <div className="mt-1">
                                <p className="text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                                  <Building2 className="w-3 h-3" />
                                  {Math.round(location.nearestBuilding.distance)}m from nearest building
                                </p>
                              </div>
                            ) : null}
                            <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                              {location.address || `${location.latitude.toFixed(4)}, ${location.longitude.toFixed(4)}`}
                            </p>
                            <p className="text-xs text-gray-500 mt-1">
                              Accuracy: ±{Math.round(location.accuracy)}m
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
                              We need your location to create a group
                            </p>
                          </div>
                          <Button
                            size="sm"
                            color="primary"
                            variant="solid"
                            onPress={getCurrentLocation}
                            isLoading={gettingLocation}
                            startContent={!gettingLocation && <MapPin className="w-4 h-4" />}
                          >
                            {gettingLocation ? "Getting Location..." : "Enable Location"}
                          </Button>
                        </>
                      )}
                    </div>
                  </div>

                  {/* Map - show immediately when location is detected */}
                  {location && (
                    <Card className="shadow-sm">
                      <CardHeader className="py-3">
                        <div className="flex items-center gap-2">
                          {location.nearestBuilding ? (
                            <>
                              <Building2 className="w-4 h-4 text-blue-500" />
                              <div className="flex-1">
                                <h3 className="font-medium text-sm">
                                  {location.nearestBuilding.isInside ? "You're Inside a Building" : "Nearest Building"}
                                </h3>
                                {location.nearestBuilding.name && (
                                  <p className="text-xs text-gray-500">{location.nearestBuilding.name}</p>
                                )}
                                {!location.nearestBuilding.isInside && (
                                  <p className="text-xs text-gray-400">
                                    {Math.round(location.nearestBuilding.distance)}m away
                                  </p>
                                )}
                              </div>
                            </>
                          ) : (
                            <>
                              <MapPin className="w-4 h-4 text-blue-500" />
                              <div className="flex-1">
                                <h3 className="font-medium text-sm">Your Location</h3>
                                <p className="text-xs text-gray-500">Group will be created here</p>
                              </div>
                            </>
                          )}
                        </div>
                      </CardHeader>
                      <CardBody className="p-0">
                        <BuildingMap
                          buildingGeometry={location.nearestBuilding?.geometry}
                          buildingCentroid={location.nearestBuilding?.centroid}
                          groupLocation={[location.latitude, location.longitude]}
                          radius={parseInt(radius)}
                          height="250px"
                        />
                      </CardBody>
                    </Card>
                  )}

                  {/* Organization Selection */}
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 block">
                      Organization (Optional)
                    </label>
                    <Select
                      placeholder="Select an organization"
                      selectedKeys={selectedOrgId ? [selectedOrgId] : []}
                      onSelectionChange={(keys) => setSelectedOrgId(Array.from(keys)[0] as string)}
                      variant="bordered"
                      classNames={{
                        trigger: "h-12 text-gray-900 dark:text-white",
                        value: "text-gray-900 dark:text-white",
                        listboxWrapper: "max-h-[300px]",
                        popoverContent: "bg-white dark:bg-gray-800",
                      }}
                      startContent={selectedOrg && (
                        <Avatar
                          src={selectedOrg.logoUrl}
                          size="sm"
                          className="w-6 h-6"
                        />
                      )}
                    >
                      {organizations.map((org) => (
                        <SelectItem
                          key={org.id}
                          value={org.id}
                          className="text-gray-900 dark:text-white"
                          startContent={
                            <Avatar
                              src={org.logoUrl}
                              size="sm"
                              className="w-6 h-6"
                            />
                          }
                        >
                          {org.name}
                        </SelectItem>
                      ))}
                    </Select>
                    <p className="text-xs text-gray-500 mt-1">
                      Add organization branding to your group
                    </p>
                  </div>

                  {/* Group Name */}
                  <Input
                    label="Group Name"
                    placeholder="e.g., Team Standup, Design Review"
                    value={groupName}
                    onValueChange={setGroupName}
                    variant="bordered"
                    size="lg"
                    isRequired
                    startContent={<Users className="w-4 h-4 text-gray-400" />}
                    classNames={{
                      inputWrapper: "border-gray-200 data-[hover=true]:border-primary",
                    }}
                  />

                  {/* Description */}
                  <Textarea
                    label="Description (Optional)"
                    placeholder="What's this group about?"
                    value={description}
                    onValueChange={setDescription}
                    variant="bordered"
                    minRows={3}
                    maxRows={5}
                    classNames={{
                      inputWrapper: "border-gray-200 data-[hover=true]:border-primary",
                    }}
                  />

                  {/* Radius */}
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 block">
                      Sharing Radius
                    </label>
                    <div className="grid grid-cols-4 gap-2">
                      {["50", "100", "200", "500"].map((value) => (
                        <Button
                          key={value}
                          variant={radius === value ? "solid" : "bordered"}
                          color={radius === value ? "primary" : "default"}
                          onPress={() => setRadius(value)}
                          size="sm"
                        >
                          {value}m
                        </Button>
                      ))}
                    </div>
                    <p className="text-xs text-gray-500 mt-2">
                      Only people within this distance can join
                    </p>
                  </div>

                  {/* Error Display */}
                  {error && (
                    <Chip color="danger" variant="flat" className="w-full">
                      <span className="text-sm">{error}</span>
                    </Chip>
                  )}

                  {/* Submit Button */}
                  <Button
                    type="submit"
                    fullWidth
                    size="lg"
                    color="primary"
                    isLoading={loading}
                    isDisabled={!location || loading || !groupName.trim()}
                    className="bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold shadow-lg hover:shadow-xl transition-all"
                    startContent={!loading && <Sparkles className="w-5 h-5" />}
                    endContent={!loading && <ArrowRight className="w-5 h-5" />}
                  >
                    Create Group
                  </Button>
                </form>
              </CardBody>
            </Card>
          </div>

          {/* Info Section */}
          <div className="space-y-6">
            {/* Selected Organization Preview */}
            {selectedOrg && (
              <Card 
                className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90"
                style={{
                  borderTop: `4px solid ${selectedOrg.brandColor}`,
                }}
              >
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <Avatar
                      src={selectedOrg.logoUrl}
                      size="md"
                    />
                    <div>
                      <h3 className="font-semibold">{selectedOrg.name}</h3>
                      <p className="text-xs text-gray-500">Organization Branding</p>
                    </div>
                  </div>
                </CardHeader>
                <CardBody className="pt-0">
                  <p className="text-xs text-gray-600">
                    Your group will display this organization's branding
                  </p>
                </CardBody>
              </Card>
            )}

            {/* Group Features */}
            <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90">
              <CardHeader>
                <h3 className="text-lg font-semibold">Group Features</h3>
              </CardHeader>
              <CardBody className="space-y-3">
                <div className="flex items-center gap-3">
                  <Clock className="w-4 h-4 text-blue-500" />
                  <div>
                    <p className="text-sm font-medium">4-Hour Duration</p>
                    <p className="text-xs text-gray-500">Extendable up to 3 times</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <MapPin className="w-4 h-4 text-green-500" />
                  <div>
                    <p className="text-sm font-medium">Location-Based</p>
                    <p className="text-xs text-gray-500">Members must be nearby to join</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Building2 className="w-4 h-4 text-purple-500" />
                  <div>
                    <p className="text-sm font-medium">Organization Support</p>
                    <p className="text-xs text-gray-500">Optional branding & features</p>
                  </div>
                </div>
              </CardBody>
            </Card>

            {/* Quick Tip */}
            <Card className="backdrop-blur-md bg-gradient-to-br from-blue-50 to-purple-50 dark:from-gray-800 dark:to-gray-700">
              <CardBody>
                <p className="text-sm font-medium mb-2">Quick Tip</p>
                <p className="text-xs text-gray-600 dark:text-gray-400">
                  For anonymous quick sharing without an account, use the{" "}
                  <Button
                    size="sm"
                    variant="light"
                    className="text-primary p-0 h-auto"
                    onPress={() => router.push("/groupup")}
                  >
                    Quick GroupUp
                  </Button>{" "}
                  feature instead.
                </p>
              </CardBody>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}