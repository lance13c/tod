"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { signOut, useSession } from "@/lib/auth-client";
import { api } from "@/providers/trpc-provider";
import Link from "next/link";

export default function DashboardPage() {
  const router = useRouter();
  const { data: session, isPending } = useSession();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedOrg, setSelectedOrg] = useState<string | null>(null);

  useEffect(() => {
    if (!isPending && !session) {
      router.push("/login");
    }
  }, [session, isPending, router]);

  // Fetch user's organizations
  const { data: organizations, refetch: refetchOrgs } = api.organization.getUserOrganizations.useQuery(
    undefined,
    { enabled: !!session }
  );

  const handleSignOut = async () => {
    await signOut();
    router.push("/login");
  };

  if (isPending) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!session) {
    return null;
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center">
              <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
            </div>
            <div className="flex items-center space-x-4">
              <Link
                href="/organizations"
                className="text-gray-500 hover:text-gray-700"
              >
                Browse Organizations
              </Link>
              <button
                onClick={handleSignOut}
                className="px-4 py-2 text-sm font-medium text-red-600 hover:text-red-700"
              >
                Sign Out
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Sidebar - User Info */}
          <div className="lg:col-span-1">
            <div className="bg-white shadow rounded-lg p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Account Information
              </h2>
              <div className="space-y-3">
                {session.user?.image ? (
                  <img
                    src={session.user.image}
                    alt=""
                    className="h-20 w-20 rounded-full mx-auto"
                  />
                ) : (
                  <div className="h-20 w-20 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-full mx-auto flex items-center justify-center text-white font-bold text-2xl">
                    {(session.user?.name || session.user?.email || "U").charAt(0).toUpperCase()}
                  </div>
                )}
                <dl className="space-y-2 text-sm">
                  {session.user?.name && (
                    <div>
                      <dt className="font-medium text-gray-500">Name</dt>
                      <dd className="text-gray-900">{session.user.name}</dd>
                    </div>
                  )}
                  <div>
                    <dt className="font-medium text-gray-500">Email</dt>
                    <dd className="text-gray-900 break-all">{session.user?.email}</dd>
                  </div>
                  <div>
                    <dt className="font-medium text-gray-500">User ID</dt>
                    <dd className="text-gray-500 text-xs font-mono break-all">
                      {session.user?.id}
                    </dd>
                  </div>
                </dl>
              </div>
            </div>

            {/* Quick Stats */}
            <div className="bg-white shadow rounded-lg p-6 mt-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">
                Statistics
              </h3>
              <dl className="space-y-3">
                <div className="flex justify-between">
                  <dt className="text-sm text-gray-500">Organizations</dt>
                  <dd className="text-sm font-medium text-gray-900">
                    {organizations?.length || 0}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-sm text-gray-500">Public</dt>
                  <dd className="text-sm font-medium text-gray-900">
                    {organizations?.filter(org => org.isPublic).length || 0}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-sm text-gray-500">Private</dt>
                  <dd className="text-sm font-medium text-gray-900">
                    {organizations?.filter(org => !org.isPublic).length || 0}
                  </dd>
                </div>
              </dl>
            </div>
          </div>

          {/* Main Content - Organizations */}
          <div className="lg:col-span-2">
            <div className="bg-white shadow rounded-lg p-6">
              <div className="flex justify-between items-center mb-6">
                <h2 className="text-lg font-semibold text-gray-900">
                  Your Organizations
                </h2>
                <button
                  onClick={() => setShowCreateModal(true)}
                  className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700"
                >
                  Create Organization
                </button>
              </div>

              {organizations && organizations.length > 0 ? (
                <div className="space-y-4">
                  {organizations.map((org) => (
                    <OrganizationCard
                      key={org.id}
                      organization={org}
                      onUpdate={refetchOrgs}
                      onEdit={() => setSelectedOrg(org.id)}
                    />
                  ))}
                </div>
              ) : (
                <div className="text-center py-12">
                  <svg
                    className="mx-auto h-12 w-12 text-gray-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
                    />
                  </svg>
                  <h3 className="mt-2 text-sm font-medium text-gray-900">
                    No organizations yet
                  </h3>
                  <p className="mt-1 text-sm text-gray-500">
                    Get started by creating your first organization.
                  </p>
                  <button
                    onClick={() => setShowCreateModal(true)}
                    className="mt-4 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700"
                  >
                    Create Organization
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Create/Edit Organization Modal */}
      {(showCreateModal || selectedOrg) && (
        <OrganizationModal
          organizationId={selectedOrg}
          onClose={() => {
            setShowCreateModal(false);
            setSelectedOrg(null);
          }}
          onSuccess={() => {
            refetchOrgs();
            setShowCreateModal(false);
            setSelectedOrg(null);
          }}
        />
      )}
    </div>
  );
}

// Organization Card Component
function OrganizationCard({ 
  organization, 
  onUpdate,
  onEdit 
}: { 
  organization: any;
  onUpdate: () => void;
  onEdit: () => void;
}) {
  const toggleVisibility = api.organization.updateOrganization.useMutation({
    onSuccess: onUpdate,
  });

  const deleteOrg = api.organization.deleteOrganization.useMutation({
    onSuccess: onUpdate,
  });

  const handleToggleVisibility = () => {
    toggleVisibility.mutate({
      id: organization.id,
      isPublic: !organization.isPublic,
    });
  };

  const handleDelete = () => {
    if (confirm(`Are you sure you want to delete ${organization.name}?`)) {
      deleteOrg.mutate({ id: organization.id });
    }
  };

  return (
    <div className="border rounded-lg p-4 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-2">
            <h3 className="text-base font-semibold text-gray-900">
              {organization.name}
            </h3>
            {organization.verified && (
              <svg className="h-4 w-4 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
            )}
            {organization.featured && (
              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800">
                Featured
              </span>
            )}
          </div>
          <p className="text-sm text-gray-500 mt-1">@{organization.slug}</p>
          {organization.description && (
            <p className="text-sm text-gray-600 mt-2">{organization.description}</p>
          )}
          <div className="flex items-center space-x-4 mt-3 text-xs text-gray-500">
            <span>{organization._count?.members || 0} members</span>
            <span>{organization.viewCount} views</span>
            {organization.industry && <span>{organization.industry}</span>}
          </div>
        </div>
        <div className="flex items-center space-x-2 ml-4">
          <button
            onClick={handleToggleVisibility}
            disabled={toggleVisibility.isPending}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              organization.isPublic ? 'bg-blue-600' : 'bg-gray-200'
            } ${toggleVisibility.isPending ? 'opacity-50' : ''}`}
          >
            <span className="sr-only">Toggle visibility</span>
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                organization.isPublic ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
          <div className="text-xs text-gray-500">
            {organization.isPublic ? 'Public' : 'Private'}
          </div>
        </div>
      </div>
      <div className="flex items-center space-x-2 mt-4">
        <Link
          href={`/org/${organization.slug}`}
          className="text-sm text-blue-600 hover:text-blue-700"
        >
          View Profile
        </Link>
        <span className="text-gray-300">•</span>
        <button
          onClick={onEdit}
          className="text-sm text-gray-600 hover:text-gray-800"
        >
          Edit
        </button>
        <span className="text-gray-300">•</span>
        <button
          onClick={handleDelete}
          disabled={deleteOrg.isPending}
          className="text-sm text-red-600 hover:text-red-700 disabled:opacity-50"
        >
          Delete
        </button>
      </div>
    </div>
  );
}

// Organization Modal Component
function OrganizationModal({ 
  organizationId, 
  onClose, 
  onSuccess 
}: { 
  organizationId: string | null;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [formData, setFormData] = useState({
    name: '',
    slug: '',
    description: '',
    industry: '',
    size: '',
    location: '',
    website: '',
    email: '',
    github: '',
    twitter: '',
    linkedin: '',
    tags: '',
    isPublic: false,
  });

  // Fetch existing organization data if editing
  const { data: existingOrg } = api.organization.getOrganization.useQuery(
    { id: organizationId! },
    { enabled: !!organizationId }
  );

  useEffect(() => {
    if (existingOrg) {
      setFormData({
        name: existingOrg.name || '',
        slug: existingOrg.slug || '',
        description: existingOrg.description || '',
        industry: existingOrg.industry || '',
        size: existingOrg.size || '',
        location: existingOrg.location || '',
        website: existingOrg.website || '',
        email: existingOrg.email || '',
        github: existingOrg.github || '',
        twitter: existingOrg.twitter || '',
        linkedin: existingOrg.linkedin || '',
        tags: existingOrg.tags || '',
        isPublic: existingOrg.isPublic || false,
      });
    }
  }, [existingOrg]);

  const createOrg = api.organization.createOrganization.useMutation({
    onSuccess,
  });

  const updateOrg = api.organization.updateOrganization.useMutation({
    onSuccess,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const payload = {
      ...formData,
      size: formData.size || undefined,
      tags: formData.tags || undefined,
      description: formData.description || undefined,
      industry: formData.industry || undefined,
      location: formData.location || undefined,
      website: formData.website || undefined,
      email: formData.email || undefined,
      github: formData.github || undefined,
      twitter: formData.twitter || undefined,
      linkedin: formData.linkedin || undefined,
    };

    if (organizationId) {
      updateOrg.mutate({ id: organizationId, ...payload });
    } else {
      createOrg.mutate(payload);
    }
  };

  // Auto-generate slug from name
  useEffect(() => {
    if (!organizationId && formData.name) {
      const slug = formData.name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-|-$/g, '');
      setFormData(prev => ({ ...prev, slug }));
    }
  }, [formData.name, organizationId]);

  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="px-6 py-4 border-b">
          <h3 className="text-lg font-semibold text-gray-900">
            {organizationId ? 'Edit Organization' : 'Create Organization'}
          </h3>
        </div>
        
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Organization Name *
              </label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="Acme Corp"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Slug *
              </label>
              <input
                type="text"
                required
                value={formData.slug}
                onChange={(e) => setFormData(prev => ({ ...prev, slug: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="acme-corp"
                pattern="[a-z0-9-]+"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              placeholder="Tell us about your organization..."
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Industry
              </label>
              <input
                type="text"
                value={formData.industry}
                onChange={(e) => setFormData(prev => ({ ...prev, industry: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="Technology"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Company Size
              </label>
              <input
                type="text"
                value={formData.size}
                onChange={(e) => setFormData(prev => ({ ...prev, size: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="10-50"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Location
              </label>
              <input
                type="text"
                value={formData.location}
                onChange={(e) => setFormData(prev => ({ ...prev, location: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="San Francisco, CA"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Contact Email
              </label>
              <input
                type="email"
                value={formData.email}
                onChange={(e) => setFormData(prev => ({ ...prev, email: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="contact@example.com"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Website
            </label>
            <input
              type="url"
              value={formData.website}
              onChange={(e) => setFormData(prev => ({ ...prev, website: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              placeholder="https://example.com"
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                GitHub
              </label>
              <input
                type="text"
                value={formData.github}
                onChange={(e) => setFormData(prev => ({ ...prev, github: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="username"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Twitter
              </label>
              <input
                type="text"
                value={formData.twitter}
                onChange={(e) => setFormData(prev => ({ ...prev, twitter: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="username"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                LinkedIn
              </label>
              <input
                type="text"
                value={formData.linkedin}
                onChange={(e) => setFormData(prev => ({ ...prev, linkedin: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="company-name"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Tags (comma-separated)
            </label>
            <input
              type="text"
              value={formData.tags}
              onChange={(e) => setFormData(prev => ({ ...prev, tags: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              placeholder="startup, saas, b2b"
            />
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="isPublic"
              checked={formData.isPublic}
              onChange={(e) => setFormData(prev => ({ ...prev, isPublic: e.target.checked }))}
              className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label htmlFor="isPublic" className="ml-2 text-sm text-gray-700">
              Make this organization public (visible in the showcase)
            </label>
          </div>

          <div className="flex justify-end space-x-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={createOrg.isPending || updateOrg.isPending}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              {createOrg.isPending || updateOrg.isPending ? 'Saving...' : (organizationId ? 'Update' : 'Create')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}