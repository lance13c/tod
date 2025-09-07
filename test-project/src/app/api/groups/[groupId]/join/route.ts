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

// POST /api/groups/[groupId]/join - Join a group by location
export async function POST(
  req: NextRequest,
  context: { params: Promise<{ groupId: string }> }
) {
  try {
    const { groupId } = await context.params;
    const body = await req.json();
    const { latitude, longitude } = body;

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

    // Find the group
    const group = await db.group.findUnique({
      where: {
        id: groupId,
        isActive: true,
      },
    });

    if (!group) {
      return NextResponse.json(
        { error: "Group not found or inactive" },
        { status: 404 }
      );
    }

    // Check if group is expired
    if (new Date(group.expiresAt) < new Date()) {
      return NextResponse.json(
        { error: "Group has expired" },
        { status: 403 }
      );
    }

    // Check if user is within the group's radius
    const distance = calculateDistance(
      latitude,
      longitude,
      group.latitude,
      group.longitude
    );

    if (distance > group.radius) {
      return NextResponse.json(
        {
          error: `You must be within ${group.radius}m of the group location to join`,
          distance: Math.round(distance),
        },
        { status: 403 }
      );
    }

    // If user is authenticated, add them as a member
    if (session?.user?.id) {
      // Check if already a member
      const existingMember = await db.groupMember.findUnique({
        where: {
          groupId_userId: {
            groupId: group.id,
            userId: session.user.id,
          },
        },
      });

      if (!existingMember) {
        await db.groupMember.create({
          data: {
            groupId: group.id,
            userId: session.user.id,
            role: "member",
            joinedAt: new Date(),
            joinedLatitude: latitude,
            joinedLongitude: longitude,
          },
        });
      }
    }

    // Return success
    return NextResponse.json({
      success: true,
      groupId: group.id,
    });
  } catch (error) {
    console.error("Failed to join group:", error);
    return NextResponse.json(
      { error: "Failed to join group" },
      { status: 500 }
    );
  }
}