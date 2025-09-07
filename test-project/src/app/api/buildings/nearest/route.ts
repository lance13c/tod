import { NextRequest, NextResponse } from 'next/server';
import { ensureDuckDBInitialized } from '@/lib/db/init';
import { findNearestBuilding } from '@/lib/geo/duckdb-building-utils';

export async function POST(req: NextRequest) {
  try {
    const { latitude, longitude, bufferMeters = 40 } = await req.json();

    if (!latitude || !longitude) {
      return NextResponse.json(
        { error: 'Latitude and longitude are required' },
        { status: 400 }
      );
    }

    console.log('Nearest building request:', { latitude, longitude, bufferMeters });

    // Initialize DuckDB if needed
    try {
      await ensureDuckDBInitialized();
    } catch (initError) {
      console.error('Failed to initialize DuckDB:', initError);
      // Continue anyway - the building search will return null if no data
    }

    // Find the nearest building
    const building = await findNearestBuilding(latitude, longitude, bufferMeters);
    
    // It's okay to return null if no building is found
    return NextResponse.json(building);
  } catch (error) {
    console.error('Error finding nearest building:', error);
    return NextResponse.json(
      { error: 'Failed to find nearest building', details: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    );
  }
}