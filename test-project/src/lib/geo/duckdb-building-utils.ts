async function spatialQuery<T = any>(query: string, params: any[] = []): Promise<T[]> {
  // Use the read-only query connection to avoid lock conflicts
  const { spatialQuery: executeQuery } = await import('@/lib/db/duckdb-query');
  return executeQuery<T>(query, params);
}

// Import coverage checking and geocoding
import { isWithinCoverage } from './coverage';
import { getBuildingAddress } from './opencage';

export interface BuildingInfo {
  id: string;
  name: string | null;
  address: string | null;
  distance: number;
  isInside: boolean;
  centroid: [number, number];
  geometry?: any;
}

/**
 * Find the nearest building to a given location using DuckDB spatial queries
 */
export async function findNearestBuilding(
  latitude: number,
  longitude: number,
  bufferMeters: number = 100
): Promise<BuildingInfo | null> {
  console.log(`Finding nearest building for location: ${latitude}, ${longitude} with buffer: ${bufferMeters}m`);
  
  // Check if location is within dataset coverage
  const coverage = await isWithinCoverage(latitude, longitude);
  if (!coverage.isWithin) {
    console.warn(`⚠️ Location outside dataset coverage: ${coverage.suggestion}`);
    console.log(`ℹ️ Try a location within the dataset, e.g., lat: ${coverage.bounds?.centerLat}, lon: ${coverage.bounds?.centerLon}`);
    return null;
  }
  
  try {
    // Create a point geometry for the user's location
    // Use ST_Distance_Sphere for accurate distance calculation in meters
    const query = `
      WITH user_location AS (
        SELECT ST_Point(?, ?) as point
      )
      SELECT 
        b.id,
        b.name,
        b.address,
        ST_Distance_Sphere(ST_Point(b.centroid_lon, b.centroid_lat), ul.point) as distance,
        ST_Contains(b.geometry, ul.point) as is_inside,
        b.centroid_lon,
        b.centroid_lat,
        ST_AsGeoJSON(b.geometry) as geometry_json
      FROM buildings b, user_location ul
      WHERE 
        -- Use bounding box for initial filtering (much faster)
        b.bbox_minx <= ? AND 
        b.bbox_maxx >= ? AND
        b.bbox_miny <= ? AND 
        b.bbox_maxy >= ?
        -- Then check actual distance using centroid
        AND ST_Distance_Sphere(ST_Point(b.centroid_lon, b.centroid_lat), ul.point) <= ?
      ORDER BY distance
      LIMIT 1
    `;

    // Convert buffer meters to approximate degrees (1 degree ≈ 111,320 meters at equator)
    // Adjust for latitude (longitude degrees get smaller as you move away from equator)
    const latBufferDegrees = bufferMeters / 111320.0;
    const lonBufferDegrees = bufferMeters / (111320.0 * Math.cos(latitude * Math.PI / 180));
    
    const results = await spatialQuery<{
      id: string;
      name: string | null;
      address: string | null;
      distance: number;
      is_inside: boolean;
      centroid_lon: number;
      centroid_lat: number;
      geometry_json: string;
    }>(query, [
      longitude, 
      latitude,
      longitude + lonBufferDegrees,  // bbox_minx <= longitude + buffer (search to the right)
      longitude - lonBufferDegrees,  // bbox_maxx >= longitude - buffer (search to the left)
      latitude + latBufferDegrees,   // bbox_miny <= latitude + buffer (search above)
      latitude - latBufferDegrees,   // bbox_maxy >= latitude - buffer (search below)
      bufferMeters
    ]);

    console.log(`DuckDB query returned ${results.length} buildings`);
    
    if (results.length === 0) {
      console.log('No buildings found within buffer distance');
      return null;
    }

    const building = results[0];
    console.log(`Found building: ID=${building.id}, distance=${Math.round(building.distance)}m, isInside=${building.is_inside}`);
    
    // Get or update the building address using OpenCage
    const address = await getBuildingAddress(
      building.id,
      building.centroid_lat,
      building.centroid_lon,
      building.address
    );
    
    // Use address as name if name is null or "unknown"
    const name = (building.name && building.name !== 'unknown' && building.name !== 'Unknown Building') 
      ? building.name 
      : address;
    
    return {
      id: building.id,
      name: name,
      address: address,
      distance: Math.round(building.distance),
      isInside: building.is_inside,
      centroid: [building.centroid_lon, building.centroid_lat],
      geometry: JSON.parse(building.geometry_json)
    };
  } catch (error) {
    console.error('Error finding nearest building with DuckDB:', error);
    return null;
  }
}

/**
 * Check if a location is inside any building
 */
export async function isLocationInBuilding(
  latitude: number,
  longitude: number,
  bufferMeters: number = 100
): Promise<{ inBuilding: boolean; building?: BuildingInfo }> {
  const building = await findNearestBuilding(latitude, longitude, bufferMeters);
  
  if (building && (building.isInside || building.distance <= bufferMeters)) {
    return {
      inBuilding: true,
      building
    };
  }
  
  return { inBuilding: false };
}

/**
 * Get all buildings within a radius
 */
export async function getBuildingsInRadius(
  latitude: number,
  longitude: number,
  radiusMeters: number
): Promise<BuildingInfo[]> {
  try {
    const query = `
      WITH user_location AS (
        SELECT ST_Point(?, ?) as point
      )
      SELECT 
        b.id,
        b.name,
        b.address,
        ST_Distance_Sphere(ST_Point(b.centroid_lon, b.centroid_lat), ul.point) as distance,
        ST_Contains(b.geometry, ul.point) as is_inside,
        b.centroid_lon,
        b.centroid_lat,
        ST_AsGeoJSON(b.geometry) as geometry_json
      FROM buildings b, user_location ul
      WHERE 
        -- Use bounding box for initial filtering
        b.bbox_minx <= ? AND 
        b.bbox_maxx >= ? AND
        b.bbox_miny <= ? AND 
        b.bbox_maxy >= ?
        -- Then check actual distance using centroid
        AND ST_Distance_Sphere(ST_Point(b.centroid_lon, b.centroid_lat), ul.point) <= ?
      ORDER BY distance
      LIMIT 100
    `;

    // Convert radius meters to approximate degrees
    // Adjust for latitude (longitude degrees get smaller as you move away from equator)
    const latRadiusDegrees = radiusMeters / 111320.0;
    const lonRadiusDegrees = radiusMeters / (111320.0 * Math.cos(latitude * Math.PI / 180));
    
    const results = await spatialQuery<{
      id: string;
      name: string | null;
      address: string | null;
      distance: number;
      is_inside: boolean;
      centroid_lon: number;
      centroid_lat: number;
      geometry_json: string;
    }>(query, [
      longitude,
      latitude,
      longitude + lonRadiusDegrees,  // bbox_minx <= longitude + radius (search to the right)
      longitude - lonRadiusDegrees,  // bbox_maxx >= longitude - radius (search to the left)
      latitude + latRadiusDegrees,   // bbox_miny <= latitude + radius (search above)
      latitude - latRadiusDegrees,   // bbox_maxy >= latitude - radius (search below)
      radiusMeters
    ]);

    // Process buildings with address lookup
    const processedBuildings = await Promise.all(
      results.map(async (building) => {
        // Get or update the building address using OpenCage
        const address = await getBuildingAddress(
          building.id,
          building.centroid_lat,
          building.centroid_lon,
          building.address
        );
        
        // Use address as name if name is null or "unknown"
        const name = (building.name && building.name !== 'unknown' && building.name !== 'Unknown Building') 
          ? building.name 
          : address;
        
        return {
          id: building.id,
          name: name,
          address: address,
          distance: Math.round(building.distance),
          isInside: building.is_inside,
          centroid: [building.centroid_lon, building.centroid_lat],
          geometry: JSON.parse(building.geometry_json)
        };
      })
    );
    
    return processedBuildings;
  } catch (error) {
    console.error('Error getting buildings in radius with DuckDB:', error);
    return [];
  }
}

/**
 * Get building by ID
 */
export async function getBuildingById(buildingId: string): Promise<BuildingInfo | null> {
  try {
    const query = `
      SELECT 
        id,
        name,
        address,
        centroid_lon,
        centroid_lat,
        ST_AsGeoJSON(geometry) as geometry_json
      FROM buildings
      WHERE id = ?
      LIMIT 1
    `;

    const results = await spatialQuery<{
      id: string;
      name: string | null;
      address: string | null;
      centroid_lon: number;
      centroid_lat: number;
      geometry_json: string;
    }>(query, [buildingId]);

    if (results.length === 0) {
      return null;
    }

    const building = results[0];
    
    // Get or update the building address using OpenCage
    const address = await getBuildingAddress(
      building.id,
      building.centroid_lat,
      building.centroid_lon,
      building.address
    );
    
    // Use address as name if name is null or "unknown"
    const name = (building.name && building.name !== 'unknown' && building.name !== 'Unknown Building') 
      ? building.name 
      : address;
    
    return {
      id: building.id,
      name: name,
      address: address,
      distance: 0,
      isInside: false,
      centroid: [building.centroid_lon, building.centroid_lat],
      geometry: JSON.parse(building.geometry_json)
    };
  } catch (error) {
    console.error('Error getting building by ID:', error);
    return null;
  }
}