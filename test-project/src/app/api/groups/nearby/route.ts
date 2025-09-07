import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { auth } from "@/lib/auth";
import { headers } from "next/headers";

// Calculate distance between two coordinates in meters
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

// POST /api/groups/nearby - Find groups near a location
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { latitude, longitude, maxDistance = 500 } = body;

    if (!latitude || !longitude) {
      return NextResponse.json(
        { error: "Location is required" },
        { status: 400 }
      );
    }

    // Get session if user is authenticated
    const session = await auth.api.getSession({
      headers: await headers(),
    });

    // Find all active groups
    const groups = await db.group.findMany({
      where: {
        isActive: true,
        expiresAt: {
          gt: new Date(),
        },
      },
      include: {
        organization: true,
        members: {
          include: {
            user: {
              select: {
                id: true,
                name: true,
                email: true,
                image: true,
              },
            },
          },
        },
        _count: {
          select: {
            files: true,
          },
        },
      },
    });

    // Filter groups by distance and check if user can join
    const nearbyGroups = groups
      .map((group) => {
        const distance = calculateDistance(
          latitude,
          longitude,
          group.latitude,
          group.longitude
        );

        // Check if user is already a member
        const isMember = session?.user?.id
          ? group.members.some((m) => m.userId === session.user.id)
          : false;

        // Check if user is within the group's radius
        const canJoin = distance <= group.radius;

        return {
          ...group,
          distance: Math.round(distance),
          canJoin,
          isMember,
        };
      })
      .filter((group) => group.distance <= maxDistance)
      .sort((a, b) => a.distance - b.distance);

    return NextResponse.json(nearbyGroups);
  } catch (error) {
    console.error("Failed to find nearby groups:", error);
    return NextResponse.json(
      { error: "Failed to find nearby groups" },
      { status: 500 }
    );
  }
}