import { Feature, FeatureCollection, Polygon } from 'geojson';

interface BuildingFeature extends Feature<Polygon> {
  properties: {
    release?: number;
    capture_dates_range?: string;
    name?: string;
    id?: string;
  };
}

interface BuildingInfo {
  feature: BuildingFeature;
  distance: number;
  isInside: boolean;
  centroid: [number, number];
}

// Check if a point is inside a polygon using ray casting algorithm
function isPointInPolygon(point: [number, number], polygon: number[][]): boolean {
  const [x, y] = point;
  let inside = false;

  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const xi = polygon[i][0];
    const yi = polygon[i][1];
    const xj = polygon[j][0];
    const yj = polygon[j][1];

    const intersect = ((yi > y) !== (yj > y))
      && (x < (xj - xi) * (y - yi) / (yj - yi) + xi);
    
    if (intersect) inside = !inside;
  }

  return inside;
}

// Calculate distance between two points using Haversine formula
function calculateDistance(
  lat1: number,
  lon1: number,
  lat2: number,
  lon2: number
): number {
  const R = 6371e3; // Earth's radius in meters
  const φ1 = (lat1 * Math.PI) / 180;
  const φ2 = (lat2 * Math.PI) / 180;
  const Δφ = ((lat2 - lat1) * Math.PI) / 180;
  const Δλ = ((lon2 - lon1) * Math.PI) / 180;

  const a =
    Math.sin(Δφ / 2) * Math.sin(Δφ / 2) +
    Math.cos(φ1) * Math.cos(φ2) * Math.sin(Δλ / 2) * Math.sin(Δλ / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));

  return R * c;
}

// Calculate the centroid of a polygon
function getPolygonCentroid(coordinates: number[][][]): [number, number] {
  let x = 0;
  let y = 0;
  const points = coordinates[0]; // Get the outer ring

  for (const point of points) {
    x += point[0];
    y += point[1];
  }

  return [x / points.length, y / points.length];
}

// Find the nearest building to a given location
export async function findNearestBuilding(
  latitude: number,
  longitude: number,
  bufferMeters: number = 40
): Promise<BuildingInfo | null> {
  try {
    // Load the GeoJSON data
    const response = await fetch('/nashville_buildings.geojson');
    const data: FeatureCollection = await response.json();
    
    let nearestBuilding: BuildingInfo | null = null;
    let minDistance = Infinity;

    for (const feature of data.features) {
      if (feature.geometry.type !== 'Polygon') continue;
      
      const buildingFeature = feature as BuildingFeature;
      const coordinates = buildingFeature.geometry.coordinates;
      const centroid = getPolygonCentroid(coordinates);
      
      // Check if point is inside the polygon
      const isInside = isPointInPolygon([longitude, latitude], coordinates[0]);
      
      // Calculate distance to centroid
      const distance = calculateDistance(
        latitude,
        longitude,
        centroid[1],
        centroid[0]
      );

      // If inside or within buffer distance
      if (isInside || distance <= bufferMeters) {
        if (distance < minDistance) {
          minDistance = distance;
          nearestBuilding = {
            feature: buildingFeature,
            distance,
            isInside,
            centroid
          };
        }
      }
    }

    return nearestBuilding;
  } catch (error) {
    console.error('Error finding nearest building:', error);
    return null;
  }
}

// Check if a location is within any building (with buffer)
export async function isLocationInBuilding(
  latitude: number,
  longitude: number,
  bufferMeters: number = 40
): Promise<{ inBuilding: boolean; building?: BuildingInfo }> {
  const building = await findNearestBuilding(latitude, longitude, bufferMeters);
  
  if (building) {
    return {
      inBuilding: true,
      building
    };
  }
  
  return { inBuilding: false };
}

// Get all buildings within a radius
export async function getBuildingsInRadius(
  latitude: number,
  longitude: number,
  radiusMeters: number
): Promise<BuildingInfo[]> {
  try {
    const response = await fetch('/nashville_buildings.geojson');
    const data: FeatureCollection = await response.json();
    
    const buildings: BuildingInfo[] = [];

    for (const feature of data.features) {
      if (feature.geometry.type !== 'Polygon') continue;
      
      const buildingFeature = feature as BuildingFeature;
      const coordinates = buildingFeature.geometry.coordinates;
      const centroid = getPolygonCentroid(coordinates);
      
      const isInside = isPointInPolygon([longitude, latitude], coordinates[0]);
      const distance = calculateDistance(
        latitude,
        longitude,
        centroid[1],
        centroid[0]
      );

      if (distance <= radiusMeters) {
        buildings.push({
          feature: buildingFeature,
          distance,
          isInside,
          centroid
        });
      }
    }

    // Sort by distance
    buildings.sort((a, b) => a.distance - b.distance);
    
    return buildings;
  } catch (error) {
    console.error('Error getting buildings in radius:', error);
    return [];
  }
}