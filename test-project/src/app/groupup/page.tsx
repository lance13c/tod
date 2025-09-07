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
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Spinner,
  Avatar,
} from "@nextui-org/react";
import {
  MapPin,
  Users,
  Clock,
  ArrowRight,
  Sparkles,
  Shield,
  AlertCircle,
  CheckCircle,
  Zap,
  Globe,
  Building2,
} from "lucide-react";

interface LocationData {
  latitude: number;
  longitude: number;
  accuracy: number;
  address?: string;
  nearestBuilding?: {
    distance: number;
    isInside: boolean;
    centroid: [number, number];
  };
}

export default function GroupUpPage() {
  const router = useRouter();
  const { data: session } = useSession();
  const [groupName, setGroupName] = useState("");
  const [description, setDescription] = useState("");
  const [radius, setRadius] = useState("100");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [location, setLocation] = useState<LocationData | null>(null);
  const [gettingLocation, setGettingLocation] = useState(false);
  const [showSuccessModal, setShowSuccessModal] = useState(false);
  const [createdGroupId, setCreatedGroupId] = useState<string | null>(null);
  const [createdGroupCode, setCreatedGroupCode] = useState<string | null>(null);

  useEffect(() => {
    // Automatically get location when page loads
    getCurrentLocation();
  }, []);

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

        // Try to get address from coordinates (optional)
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
              bufferMeters: 40
            })
          });
          
          if (response.ok) {
            const building = await response.json();
            if (building) {
              locationData.nearestBuilding = {
                distance: building.distance,
                isInside: building.isInside,
                centroid: building.centroid,
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
        let errorMessage = "Failed to get location: ";
        switch(error.code) {
          case error.PERMISSION_DENIED:
            errorMessage += "Location access was denied. Please enable location permissions in your browser settings.";
            break;
          case error.POSITION_UNAVAILABLE:
            errorMessage += "Location information is unavailable. Please check your device's location settings.";
            break;
          case error.TIMEOUT:
            errorMessage += "Location request timed out. Please try again or check your internet connection.";
            break;
          default:
            errorMessage += error.message;
        }
        setError(errorMessage);
        setGettingLocation(false);
      },
      {
        enableHighAccuracy: true,
        timeout: 30000, // Increase timeout to 30 seconds
        maximumAge: 60000, // Accept cached position up to 1 minute old
      }
    );
  };

  const handleCreateGroup = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!location) {
      setError("Please enable location access to create a group");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch("/api/groups", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: groupName || "Quick Group",
          description,
          latitude: location.latitude,
          longitude: location.longitude,
          radius: parseInt(radius),
          organizationId: null, // No organization for anonymous groups
          isAnonymous: !session,
        }),
      });
      
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to create group");
      }
      
      const group = await response.json();
      setCreatedGroupId(group.id);
      setCreatedGroupCode(group.code);
      setShowSuccessModal(true);
      
      // If user is logged in, redirect to the group page after a delay
      if (session) {
        setTimeout(() => {
          router.push(`/groups/${group.id}`);
        }, 2000);
      }
    } catch (err: any) {
      setError(err.message || "Failed to create group");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      {/* Header */}
      <div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-md shadow-sm border-b dark:border-gray-700">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl">
                <Zap className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                  Quick GroupUp
                </h1>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Start sharing instantly, no account needed
                </p>
              </div>
            </div>
            {!session && (
              <Button
                variant="light"
                size="sm"
                onPress={() => router.push("/login")}
                startContent={<Shield className="w-4 h-4" />}
              >
                Sign In
              </Button>
            )}
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Form Section */}
          <div className="lg:col-span-2">
            <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90 shadow-xl">
              <CardHeader className="pb-2">
                <h2 className="text-xl font-semibold">Create Your Group</h2>
              </CardHeader>
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
                              GPS Accuracy: ±{Math.round(location.accuracy)}m
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
                            {error && (
                              <p className="text-xs text-red-500 mt-1">
                                {error}
                              </p>
                            )}
                          </div>
                          <div className="flex flex-col gap-2">
                            <Button
                              size="sm"
                              color="primary"
                              variant="flat"
                              onPress={getCurrentLocation}
                              startContent={<MapPin className="w-4 h-4" />}
                            >
                              Try Again
                            </Button>
                            <Button
                              size="sm"
                              variant="light"
                              onPress={() => {
                                // Use a default Nashville location as fallback
                                const defaultLocation: LocationData = {
                                  latitude: 36.1627,
                                  longitude: -86.7816,
                                  accuracy: 100,
                                  address: "Nashville, TN (Default Location)"
                                };
                                setLocation(defaultLocation);
                                setError(null);
                              }}
                            >
                              Use Default
                            </Button>
                          </div>
                        </>
                      )}
                    </div>
                  </div>

                  {/* Group Name */}
                  <Input
                    label="Group Name (Optional)"
                    placeholder="e.g., Coffee Break, Team Meetup"
                    value={groupName}
                    onValueChange={setGroupName}
                    variant="bordered"
                    size="lg"
                    startContent={<Users className="w-4 h-4 text-gray-400" />}
                    description="Leave empty for a random name"
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
                    minRows={2}
                    maxRows={4}
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
                    isDisabled={!location || loading}
                    className="bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold shadow-lg hover:shadow-xl transition-all"
                    startContent={!loading && <Sparkles className="w-5 h-5" />}
                    endContent={!loading && <ArrowRight className="w-5 h-5" />}
                  >
                    Create Group & Start Sharing
                  </Button>

                  {!session && (
                    <div className="text-center">
                      <p className="text-sm text-gray-500">
                        Want to save your groups?{" "}
                        <Button
                          variant="light"
                          size="sm"
                          onPress={() => router.push("/signup")}
                          className="text-primary"
                        >
                          Create an account
                        </Button>
                      </p>
                    </div>
                  )}
                </form>
              </CardBody>
            </Card>
          </div>

          {/* Info Section */}
          <div className="space-y-6">
            {/* How it Works */}
            <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90">
              <CardHeader>
                <h3 className="text-lg font-semibold">How it Works</h3>
              </CardHeader>
              <CardBody className="space-y-4">
                <div className="flex gap-3">
                  <div className="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center flex-shrink-0">
                    <span className="text-sm font-bold text-blue-600 dark:text-blue-400">1</span>
                  </div>
                  <div>
                    <p className="font-medium text-sm">Create a group</p>
                    <p className="text-xs text-gray-500">Set your location and radius</p>
                  </div>
                </div>
                <div className="flex gap-3">
                  <div className="w-8 h-8 rounded-full bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center flex-shrink-0">
                    <span className="text-sm font-bold text-purple-600 dark:text-purple-400">2</span>
                  </div>
                  <div>
                    <p className="font-medium text-sm">Share the code</p>
                    <p className="text-xs text-gray-500">Others nearby can join instantly</p>
                  </div>
                </div>
                <div className="flex gap-3">
                  <div className="w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center flex-shrink-0">
                    <span className="text-sm font-bold text-green-600 dark:text-green-400">3</span>
                  </div>
                  <div>
                    <p className="font-medium text-sm">Start sharing</p>
                    <p className="text-xs text-gray-500">Exchange files directly, no server storage</p>
                  </div>
                </div>
              </CardBody>
            </Card>

            {/* Features */}
            <Card className="backdrop-blur-md bg-white/90 dark:bg-gray-800/90">
              <CardHeader>
                <h3 className="text-lg font-semibold">Quick Features</h3>
              </CardHeader>
              <CardBody className="space-y-3">
                <div className="flex items-center gap-3">
                  <Clock className="w-4 h-4 text-blue-500" />
                  <div>
                    <p className="text-sm font-medium">4-Hour Duration</p>
                    <p className="text-xs text-gray-500">Auto-expires for privacy</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Globe className="w-4 h-4 text-green-500" />
                  <div>
                    <p className="text-sm font-medium">No Account Needed</p>
                    <p className="text-xs text-gray-500">Start sharing instantly</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Shield className="w-4 h-4 text-purple-500" />
                  <div>
                    <p className="text-sm font-medium">P2P Transfer</p>
                    <p className="text-xs text-gray-500">Direct device-to-device</p>
                  </div>
                </div>
              </CardBody>
            </Card>
          </div>
        </div>
      </div>

      {/* Success Modal */}
      <Modal 
        isOpen={showSuccessModal} 
        onClose={() => setShowSuccessModal(false)}
        placement="center"
      >
        <ModalContent>
          <ModalHeader className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <CheckCircle className="w-6 h-6 text-green-500" />
              Group Created Successfully!
            </div>
          </ModalHeader>
          <ModalBody>
            <div className="space-y-4">
              <p className="text-sm text-gray-600">
                Your group has been created and is now active for the next 4 hours.
              </p>
              <div className="bg-gray-100 dark:bg-gray-700 rounded-lg p-4">
                <p className="text-xs text-gray-500 mb-2">Group Code</p>
                <p className="text-2xl font-mono font-bold text-center">
                  {createdGroupCode || createdGroupId?.slice(0, 6).toUpperCase() || "ABC123"}
                </p>
              </div>
              <p className="text-xs text-gray-500 text-center">
                Share this code with people nearby to let them join
              </p>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button 
              color="primary" 
              onPress={() => {
                setShowSuccessModal(false);
                if (!session) {
                  // For anonymous users, show the code and let them share
                  navigator.clipboard.writeText(createdGroupCode || createdGroupId?.slice(0, 6).toUpperCase() || "");
                }
              }}
            >
              {session ? "Go to Group" : "Copy Code"}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
}