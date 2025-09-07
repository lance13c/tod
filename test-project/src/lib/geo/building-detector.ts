import type { Feature, Polygon, Point } from 'geojson';

/**
 * Detects if a point is inside a building polygon using ray-casting algorithm
 */
export function isPointInPolygon(point: [number, number], polygon: number[][]): boolean {
  const [x, y] = point;
  let inside = false;

  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const xi = polygon[i][0];
    const yi = polygon[i][1];
    const xj = polygon[j][0];
    const yj = polygon[j][1];

    const intersect = ((yi > y) !== (yj > y)) && 
                     (x < (xj - xi) * (y - yi) / (yj - yi) + xi);
    
    if (intersect) inside = !inside;
  }

  return inside;
}

/**
 * Calculate bounding box for a polygon
 */
export function calculateBoundingBox(polygon: number[][]): {
  minLat: number;
  maxLat: number;
  minLng: number;
  maxLng: number;
} {
  let minLat = Infinity;
  let maxLat = -Infinity;
  let minLng = Infinity;
  let maxLng = -Infinity;

  for (const [lng, lat] of polygon) {
    minLat = Math.min(minLat, lat);
    maxLat = Math.max(maxLat, lat);
    minLng = Math.min(minLng, lng);
    maxLng = Math.max(maxLng, lng);
  }

  return { minLat, maxLat, minLng, maxLng };
}

/**
 * Quick check if point might be in polygon using bounding box
 */
export function isPointInBoundingBox(
  point: [number, number],
  bbox: { minLat: number; maxLat: number; minLng: number; maxLng: number }
): boolean {
  const [lng, lat] = point;
  return lat >= bbox.minLat && lat <= bbox.maxLat && 
         lng >= bbox.minLng && lng <= bbox.maxLng;
}

/**
 * Find building containing the given coordinates
 */
export async function findBuildingAtLocation(
  latitude: number,
  longitude: number,
  buildings: Array<{
    id: string;
    polygon: string;
    bbox: string | null;
    name: string | null;
  }>
): Promise<{
  id: string;
  name: string | null;
} | null> {
  const point: [number, number] = [longitude, latitude];

  for (const building of buildings) {
    // Quick bounding box check first
    if (building.bbox) {
      const bbox = JSON.parse(building.bbox);
      if (!isPointInBoundingBox(point, bbox)) {
        continue;
      }
    }

    // Detailed polygon check
    const polygonData = JSON.parse(building.polygon);
    let coordinates: number[][];

    // Handle different GeoJSON structures
    if (polygonData.type === 'Polygon') {
      coordinates = polygonData.coordinates[0];
    } else if (polygonData.coordinates) {
      coordinates = polygonData.coordinates[0];
    } else if (Array.isArray(polygonData)) {
      coordinates = polygonData;
    } else {
      continue;
    }

    if (isPointInPolygon(point, coordinates)) {
      return {
        id: building.id,
        name: building.name
      };
    }
  }

  return null;
}

/**
 * Calculate distance between two coordinates in meters using Haversine formula
 */
export function calculateDistance(
  lat1: number,
  lon1: number,
  lat2: number,
  lon2: number
): number {
  const R = 6371e3; // Earth's radius in meters
  const φ1 = lat1 * Math.PI / 180;
  const φ2 = lat2 * Math.PI / 180;
  const Δφ = (lat2 - lat1) * Math.PI / 180;
  const Δλ = (lon2 - lon1) * Math.PI / 180;

  const a = Math.sin(Δφ / 2) * Math.sin(Δφ / 2) +
            Math.cos(φ1) * Math.cos(φ2) *
            Math.sin(Δλ / 2) * Math.sin(Δλ / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));

  return R * c;
}

/**
 * Check if user is within geo-lock radius of a session
 */
export function isWithinGeoLock(
  userLat: number,
  userLon: number,
  sessionLat: number,
  sessionLon: number,
  radiusMeters: number
): boolean {
  const distance = calculateDistance(userLat, userLon, sessionLat, sessionLon);
  return distance <= radiusMeters;
}

/**
 * Generate a random 6-character alphanumeric code
 */
export function generateSessionCode(): string {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let code = '';
  for (let i = 0; i < 6; i++) {
    code += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return code;
}

/**
 * Parse US Building Footprints GeoJSON feature
 */
export function parseBuildingFootprint(feature: Feature): {
  polygon: number[][];
  properties: Record<string, any>;
} | null {
  if (feature.geometry?.type !== 'Polygon') {
    return null;
  }

  const polygon = feature.geometry as Polygon;
  return {
    polygon: polygon.coordinates[0],
    properties: feature.properties || {}
  };
}