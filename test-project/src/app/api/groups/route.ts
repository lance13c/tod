import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { auth } from "@/lib/auth";
import { headers } from "next/headers";
import { ensureDuckDBInitialized } from "@/lib/db/init";
import { findNearestBuilding } from "@/lib/geo/duckdb-building-utils";

// Calculate expiry time (4 hours from now)
function calculateExpiry(): Date {
  const now = new Date();
  now.setHours(now.getHours() + 4);
  return now;
}

// POST /api/groups - Create a new group
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const {
      name,
      description,
      latitude,
      longitude,
      radius,
      organizationId,
      isAnonymous,
    } = body;

    // Get session if user is authenticated
    const session = await auth.api.getSession({
      headers: await headers(),
    });

    // Initialize DuckDB for spatial queries
    await ensureDuckDBInitialized();

    // Find nearest building if location is provided
    let buildingId = null;
    if (latitude && longitude) {
      const nearestBuilding = await findNearestBuilding(latitude, longitude, 40);
      if (nearestBuilding && nearestBuilding.isInside) {
        // If user is inside a building, associate the group with it
        buildingId = nearestBuilding.id;
        
        // First check if building exists in our database, if not create it
        const existingBuilding = await db.building.findUnique({
          where: { id: buildingId }
        });
        
        if (!existingBuilding) {
          await db.building.create({
            data: {
              id: buildingId,
              name: nearestBuilding.name,
              address: nearestBuilding.address,
              polygon: JSON.stringify(nearestBuilding.geometry),
              area: 0, // We can calculate this if needed
            }
          });
        }
      }
    }

    // Create storage folder UUID
    const storageFolder = crypto.randomUUID();

    // Create the group
    const group = await db.group.create({
      data: {
        name: name || "Quick Group",
        description,
        organizationId: organizationId || null,
        creatorId: session?.user?.id || null,
        buildingId,
        expiresAt: calculateExpiry(),
        storageFolder,
        isActive: true,
        latitude,
        longitude,
        radius: radius || 100,
      },
    });

    // If user is authenticated, add them as the group creator member
    if (session?.user?.id) {
      await db.groupMember.create({
        data: {
          groupId: group.id,
          userId: session.user.id,
          role: "creator",
          joinedAt: new Date(),
        },
      });
    }

    // Return the created group
    return NextResponse.json(group);
  } catch (error) {
    console.error("Failed to create group:", error);
    return NextResponse.json(
      { error: "Failed to create group" },
      { status: 500 }
    );
  }
}

// GET /api/groups - Get groups (for authenticated users)
export async function GET(req: NextRequest) {
  try {
    const session = await auth.api.getSession({
      headers: await headers(),
    });

    if (!session?.user?.id) {
      return NextResponse.json(
        { error: "Authentication required" },
        { status: 401 }
      );
    }

    // Get all groups the user is a member of
    const groupMembers = await db.groupMember.findMany({
      where: {
        userId: session.user.id,
      },
      include: {
        group: {
          include: {
            organization: true,
            members: true,
            files: true,
          },
        },
      },
      orderBy: {
        joinedAt: "desc",
      },
    });

    const groups = groupMembers.map((gm) => ({
      ...gm.group,
      role: gm.role,
      joinedAt: gm.joinedAt,
    }));

    return NextResponse.json(groups);
  } catch (error) {
    console.error("Failed to fetch groups:", error);
    return NextResponse.json(
      { error: "Failed to fetch groups" },
      { status: 500 }
    );
  }
}