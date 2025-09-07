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

// POST /api/groups/join - Join a group by code
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { code, latitude, longitude } = body;

    if (!code || !latitude || !longitude) {
      return NextResponse.json(
        { error: "Code and location are required" },
        { status: 400 }
      );
    }

    // Find group by code (search for groups starting with the code)
    const groups = await db.group.findMany({
      where: {
        isActive: true,
        AND: [
          {
            expiresAt: {
              gt: new Date(),
            },
          },
        ],
      },
    });

    // Find matching group by code
    const group = groups.find(
      (g) => g.id.slice(0, 6).toUpperCase() === code.toUpperCase()
    );

    if (!group) {
      return NextResponse.json(
        { error: "Invalid or expired group code" },
        { status: 404 }
      );
    }

    // Check if user is within the group's radius
    if (group.latitude && group.longitude && group.radius) {
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
    }

    // Get session if user is authenticated
    const session = await auth.api.getSession({
      headers: await headers(),
    });

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
          },
        });
      }
    }

    // Return the group details
    const fullGroup = await db.group.findUnique({
      where: { id: group.id },
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
        files: true,
      },
    });

    return NextResponse.json({
      ...fullGroup,
      code: code.toUpperCase(),
    });
  } catch (error) {
    console.error("Failed to join group:", error);
    return NextResponse.json(
      { error: "Failed to join group" },
      { status: 500 }
    );
  }
}