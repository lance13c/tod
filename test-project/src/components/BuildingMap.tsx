"use client";

import { useEffect, useRef } from "react";
import dynamic from "next/dynamic";
import type { Map as LeafletMap } from "leaflet";

// Dynamic imports for Leaflet to avoid SSR issues
const MapContainer = dynamic(
  () => import("react-leaflet").then((mod) => mod.MapContainer),
  { ssr: false }
);
const TileLayer = dynamic(
  () => import("react-leaflet").then((mod) => mod.TileLayer),
  { ssr: false }
);
const Polygon = dynamic(
  () => import("react-leaflet").then((mod) => mod.Polygon),
  { ssr: false }
);
const Marker = dynamic(
  () => import("react-leaflet").then((mod) => mod.Marker),
  { ssr: false }
);
const Popup = dynamic(
  () => import("react-leaflet").then((mod) => mod.Popup),
  { ssr: false }
);
const Circle = dynamic(
  () => import("react-leaflet").then((mod) => mod.Circle),
  { ssr: false }
);

interface BuildingMapProps {
  buildingGeometry?: any;
  buildingCentroid?: [number, number];
  groupLocation?: [number, number];
  radius?: number;
  height?: string;
}

export default function BuildingMap({
  buildingGeometry,
  buildingCentroid,
  groupLocation,
  radius = 100,
  height = "400px",
}: BuildingMapProps) {
  const mapRef = useRef<LeafletMap | null>(null);

  // Debug logging
  useEffect(() => {
    console.log("🗺️ BuildingMap props:", {
      hasGeometry: !!buildingGeometry,
      geometryType: buildingGeometry?.type,
      hasCentroid: !!buildingCentroid,
      centroid: buildingCentroid,
      hasGroupLocation: !!groupLocation,
      groupLocation: groupLocation,
      radius: radius
    });
  }, [buildingGeometry, buildingCentroid, groupLocation, radius]);

  useEffect(() => {
    // Fix for Leaflet icon issues in Next.js
    if (typeof window !== "undefined") {
      const L = require("leaflet");
      delete (L.Icon.Default.prototype as any)._getIconUrl;
      L.Icon.Default.mergeOptions({
        iconRetinaUrl: "/leaflet/marker-icon-2x.png",
        iconUrl: "/leaflet/marker-icon.png",
        shadowUrl: "/leaflet/marker-shadow.png",
      });
    }
  }, []);

  // Center point - prefer building centroid, fallback to group location
  // Note: buildingCentroid comes as [lng, lat] from backend, need to swap for Leaflet
  const center = buildingCentroid 
    ? [buildingCentroid[1], buildingCentroid[0]] 
    : groupLocation || [36.1627, -86.7816]; // Default to Nashville

  // Convert GeoJSON polygon to Leaflet format
  const getPolygonPositions = () => {
    if (!buildingGeometry || buildingGeometry.type !== "Polygon") return null;
    
    // GeoJSON uses [lng, lat], Leaflet uses [lat, lng]
    return buildingGeometry.coordinates[0].map((coord: number[]) => [coord[1], coord[0]]);
  };

  const polygonPositions = getPolygonPositions();

  return (
    <div style={{ height, width: "100%" }} className="rounded-lg overflow-hidden">
      <MapContainer
        center={center as [number, number]}
        zoom={18}
        style={{ height: "100%", width: "100%" }}
        ref={mapRef}
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        
        {/* Building polygon */}
        {polygonPositions && (
          <Polygon
            positions={polygonPositions}
            pathOptions={{
              color: "#3B82F6",
              weight: 2,
              opacity: 0.8,
              fillColor: "#60A5FA",
              fillOpacity: 0.3,
            }}
          >
            <Popup>Building Boundary</Popup>
          </Polygon>
        )}
        
        {/* Group location marker and radius circle */}
        {groupLocation && (
          <>
            <Circle
              center={groupLocation as [number, number]}
              radius={radius}
              pathOptions={{
                color: "#10B981",
                weight: 2,
                opacity: 0.6,
                fillColor: "#10B981",
                fillOpacity: 0.1,
              }}
            />
            <Marker position={groupLocation as [number, number]}>
              <Popup>
                Your Location
                <br />
                Group Radius: {radius}m
              </Popup>
            </Marker>
          </>
        )}
      </MapContainer>
    </div>
  );
}