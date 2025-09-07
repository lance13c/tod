/**
 * OpenCage Geocoding API integration for reverse geocoding
 */

interface OpenCageResponse {
  results: Array<{
    formatted: string;
    components: {
      house_number?: string;
      road?: string;
      suburb?: string;
      city?: string;
      state?: string;
      postcode?: string;
      country?: string;
      building?: string;
      neighbourhood?: string;
    };
    geometry: {
      lat: number;
      lng: number;
    };
  }>;
  status: {
    code: number;
    message: string;
  };
}

/**
 * Get a formatted address from coordinates using OpenCage API
 */
export async function reverseGeocode(
  latitude: number,
  longitude: number
): Promise<{ address: string; components: any } | null> {
  const apiKey = process.env.OPENCAGE_API_KEY;
  
  if (!apiKey) {
    console.warn('OpenCage API key not configured');
    return null;
  }

  try {
    const url = `https://api.opencagedata.com/geocode/v1/json?key=${apiKey}&q=${latitude}%2C${longitude}&pretty=1&no_annotations=1`;
    
    const response = await fetch(url);
    
    if (!response.ok) {
      console.error('OpenCage API error:', response.status);
      return null;
    }

    const data: OpenCageResponse = await response.json();
    
    if (data.results && data.results.length > 0) {
      const result = data.results[0];
      
      // Build a clean address from components
      const components = result.components;
      let address = '';
      
      // Priority: road name is most important for session naming
      if (components.road) {
        address = components.road;
      } else if (components.building) {
        address = components.building;
      } else if (components.neighbourhood) {
        address = components.neighbourhood;
      } else if (components.suburb) {
        address = components.suburb;
      } else {
        // Fallback to formatted address
        address = result.formatted;
        // Clean up the address
        address = address.split(',')[0]; // Take first part before comma
      }
      
      console.log(`Reverse geocoded (${latitude}, ${longitude}) to: ${address}`);
      
      return {
        address,
        components
      };
    }
    
    return null;
  } catch (error) {
    console.error('Error calling OpenCage API:', error);
    return null;
  }
}

/**
 * Get or cache building address
 */
export async function getBuildingAddress(
  buildingId: string,
  latitude: number,
  longitude: number,
  currentAddress: string | null
): Promise<string> {
  // If we already have a good address (not null, not "unknown"), use it
  if (currentAddress && 
      currentAddress !== 'unknown' && 
      currentAddress !== 'Unknown Building' &&
      currentAddress !== '') {
    return currentAddress;
  }
  
  // Try to get address from OpenCage
  const geocodeResult = await reverseGeocode(latitude, longitude);
  
  if (geocodeResult && geocodeResult.address) {
    // Update the building in the database with the new address
    try {
      const { initializeDuckDB } = await import('@/lib/db/duckdb');
      const db = await initializeDuckDB();
      
      await db.run(
        'UPDATE buildings SET address = ?, name = ? WHERE id = ?',
        geocodeResult.address,
        geocodeResult.address, // Also use as name if name is null
        buildingId
      );
      
      console.log(`Updated building ${buildingId} with address: ${geocodeResult.address}`);
      
      return geocodeResult.address;
    } catch (error) {
      console.error('Error updating building address:', error);
      return geocodeResult.address; // Still return the address even if update fails
    }
  }
  
  // Fallback to "Building" if all else fails
  return 'Building';
}