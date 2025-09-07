-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_Group" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "organizationId" TEXT,
    "creatorId" TEXT,
    "latitude" REAL NOT NULL,
    "longitude" REAL NOT NULL,
    "radius" INTEGER NOT NULL DEFAULT 100,
    "buildingId" TEXT,
    "expiresAt" DATETIME NOT NULL,
    "extendedCount" INTEGER NOT NULL DEFAULT 0,
    "maxExtensions" INTEGER NOT NULL DEFAULT 3,
    "storageFolder" TEXT NOT NULL,
    "isActive" BOOLEAN NOT NULL DEFAULT true,
    "isArchived" BOOLEAN NOT NULL DEFAULT false,
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" DATETIME NOT NULL,
    CONSTRAINT "Group_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "Organization" ("id") ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT "Group_creatorId_fkey" FOREIGN KEY ("creatorId") REFERENCES "User" ("id") ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT "Group_buildingId_fkey" FOREIGN KEY ("buildingId") REFERENCES "Building" ("id") ON DELETE SET NULL ON UPDATE CASCADE
);
INSERT INTO "new_Group" ("buildingId", "createdAt", "creatorId", "description", "expiresAt", "extendedCount", "id", "isActive", "isArchived", "latitude", "longitude", "maxExtensions", "name", "organizationId", "radius", "storageFolder", "updatedAt") SELECT "buildingId", "createdAt", "creatorId", "description", "expiresAt", "extendedCount", "id", "isActive", "isArchived", "latitude", "longitude", "maxExtensions", "name", "organizationId", "radius", "storageFolder", "updatedAt" FROM "Group";
DROP TABLE "Group";
ALTER TABLE "new_Group" RENAME TO "Group";
CREATE UNIQUE INDEX "Group_storageFolder_key" ON "Group"("storageFolder");
CREATE INDEX "Group_organizationId_idx" ON "Group"("organizationId");
CREATE INDEX "Group_creatorId_idx" ON "Group"("creatorId");
CREATE INDEX "Group_expiresAt_idx" ON "Group"("expiresAt");
CREATE INDEX "Group_isActive_idx" ON "Group"("isActive");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
