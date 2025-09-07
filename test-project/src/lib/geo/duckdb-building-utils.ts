async function spatialQuery<T = any>(query: string, params: any[] = []): Promise<T[]> {
  const { spatialQuery: executeQuery } = await import('@/lib/db/duckdb');
  return executeQuery<T>(query, params);
}

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
  bufferMeters: number = 40
): Promise<BuildingInfo | null> {
  console.log(`Finding nearest building for location: ${latitude}, ${longitude} with buffer: ${bufferMeters}m`);
  
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
        ST_Distance_Sphere(b.geometry, ul.point) as distance,
        ST_Contains(b.geometry, ul.point) as is_inside,
        b.centroid_lon,
        b.centroid_lat,
        ST_AsGeoJSON(b.geometry) as geometry_json
      FROM buildings b, user_location ul
      WHERE 
        -- Use bounding box for initial filtering (much faster)
        b.bbox_minx <= ? + (? / 111320.0) AND 
        b.bbox_maxx >= ? - (? / 111320.0) AND
        b.bbox_miny <= ? + (? / 111320.0) AND 
        b.bbox_maxy >= ? - (? / 111320.0)
        -- Then check actual distance
        AND ST_Distance_Sphere(b.geometry, ul.point) <= ?
      ORDER BY distance
      LIMIT 1
    `;

    // Convert buffer meters to approximate degrees (1 degree ≈ 111,320 meters at equator)
    const bufferDegrees = bufferMeters / 111320.0;
    
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
      longitude, bufferDegrees,
      longitude, bufferDegrees,
      latitude, bufferDegrees,
      latitude, bufferDegrees,
      bufferMeters
    ]);

    console.log(`DuckDB query returned ${results.length} buildings`);
    
    if (results.length === 0) {
      console.log('No buildings found within buffer distance');
      return null;
    }

    const building = results[0];
    console.log(`Found building: ID=${building.id}, distance=${Math.round(building.distance)}m, isInside=${building.is_inside}`);
    return {
      id: building.id,
      name: building.name,
      address: building.address,
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
  bufferMeters: number = 40
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
        ST_Distance_Sphere(b.geometry, ul.point) as distance,
        ST_Contains(b.geometry, ul.point) as is_inside,
        b.centroid_lon,
        b.centroid_lat,
        ST_AsGeoJSON(b.geometry) as geometry_json
      FROM buildings b, user_location ul
      WHERE 
        -- Use bounding box for initial filtering
        b.bbox_minx <= ? + (? / 111320.0) AND 
        b.bbox_maxx >= ? - (? / 111320.0) AND
        b.bbox_miny <= ? + (? / 111320.0) AND 
        b.bbox_maxy >= ? - (? / 111320.0)
        -- Then check actual distance
        AND ST_Distance_Sphere(b.geometry, ul.point) <= ?
      ORDER BY distance
      LIMIT 100
    `;

    const radiusDegrees = radiusMeters / 111320.0;
    
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
      longitude, radiusDegrees,
      longitude, radiusDegrees,
      latitude, radiusDegrees,
      latitude, radiusDegrees,
      radiusMeters
    ]);

    return results.map(building => ({
      id: building.id,
      name: building.name,
      address: building.address,
      distance: Math.round(building.distance),
      isInside: building.is_inside,
      centroid: [building.centroid_lon, building.centroid_lat],
      geometry: JSON.parse(building.geometry_json)
    }));
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
    return {
      id: building.id,
      name: building.name,
      address: building.address,
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