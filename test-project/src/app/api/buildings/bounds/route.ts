import { NextRequest, NextResponse } from 'next/server';
import { getDatasetBounds, getSampleLocation } from '@/lib/geo/coverage';

export async function GET(req: NextRequest) {
  try {
    const bounds = await getDatasetBounds();
    const sampleLocation = await getSampleLocation();
    
    if (!bounds) {
      return NextResponse.json({
        error: 'No dataset loaded',
        message: 'Buildings dataset is not loaded. Please restart the server or load data manually.',
      }, { status: 404 });
    }
    
    return NextResponse.json({
      bounds,
      sampleLocation,
      coverage: {
        description: 'Nashville buildings dataset',
        totalArea: `${((bounds.maxLon - bounds.minLon) * (bounds.maxLat - bounds.minLat) * 12321).toFixed(2)} km²`, // Rough approximation
        center: {
          latitude: bounds.centerLat,
          longitude: bounds.centerLon
        }
      },
      usage: {
        note: 'Use coordinates within these bounds for building detection',
        example: {
          latitude: sampleLocation?.latitude || bounds.centerLat,
          longitude: sampleLocation?.longitude || bounds.centerLon
        }
      }
    });
  } catch (error) {
    console.error('Error getting dataset bounds:', error);
    return NextResponse.json(
      { 
        error: 'Failed to get dataset bounds',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}