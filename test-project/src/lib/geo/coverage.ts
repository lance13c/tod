import { getDuckDB } from '@/lib/db/duckdb';

export interface DatasetBounds {
  minLon: number;
  maxLon: number;
  minLat: number;
  maxLat: number;
  centerLon: number;
  centerLat: number;
}

/**
 * Get the bounds of the loaded dataset
 */
export async function getDatasetBounds(): Promise<DatasetBounds | null> {
  try {
    const db = await getDuckDB();
    
    const result = await db.all(`
      SELECT 
        CAST(MIN(bbox_minx) AS DOUBLE) as min_lon,
        CAST(MAX(bbox_maxx) AS DOUBLE) as max_lon,
        CAST(MIN(bbox_miny) AS DOUBLE) as min_lat,
        CAST(MAX(bbox_maxy) AS DOUBLE) as max_lat,
        CAST(AVG(centroid_lon) AS DOUBLE) as center_lon,
        CAST(AVG(centroid_lat) AS DOUBLE) as center_lat
      FROM buildings
      WHERE bbox_minx IS NOT NULL
    `);
    
    if (result.length === 0 || !result[0].min_lon) {
      return null;
    }
    
    return {
      minLon: result[0].min_lon,
      maxLon: result[0].max_lon,
      minLat: result[0].min_lat,
      maxLat: result[0].max_lat,
      centerLon: result[0].center_lon,
      centerLat: result[0].center_lat,
    };
  } catch (error) {
    console.error('Error getting dataset bounds:', error);
    return null;
  }
}

/**
 * Check if coordinates are within the dataset coverage area
 */
export async function isWithinCoverage(latitude: number, longitude: number): Promise<{
  isWithin: boolean;
  bounds?: DatasetBounds;
  distance?: number;
  suggestion?: string;
}> {
  const bounds = await getDatasetBounds();
  
  if (!bounds) {
    return {
      isWithin: false,
      suggestion: 'No dataset loaded. Please ensure buildings data is loaded.'
    };
  }
  
  const isWithin = 
    longitude >= bounds.minLon &&
    longitude <= bounds.maxLon &&
    latitude >= bounds.minLat &&
    latitude <= bounds.maxLat;
  
  if (!isWithin) {
    // Calculate approximate distance to dataset center
    const distance = Math.sqrt(
      Math.pow((longitude - bounds.centerLon) * 111.32, 2) + // km per degree longitude at ~36° latitude
      Math.pow((latitude - bounds.centerLat) * 110.54, 2)    // km per degree latitude
    );
    
    return {
      isWithin: false,
      bounds,
      distance: Math.round(distance),
      suggestion: `Location is outside dataset coverage (approximately ${Math.round(distance)}km away). Dataset covers: Longitude ${bounds.minLon.toFixed(3)} to ${bounds.maxLon.toFixed(3)}, Latitude ${bounds.minLat.toFixed(3)} to ${bounds.maxLat.toFixed(3)}`
    };
  }
  
  return {
    isWithin: true,
    bounds
  };
}

/**
 * Get a sample location within the dataset bounds
 */
export async function getSampleLocation(): Promise<{ latitude: number; longitude: number } | null> {
  const bounds = await getDatasetBounds();
  
  if (!bounds) {
    return null;
  }
  
  return {
    latitude: bounds.centerLat,
    longitude: bounds.centerLon
  };
}