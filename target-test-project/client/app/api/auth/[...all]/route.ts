import { NextRequest, NextResponse } from 'next/server'

// Proxy all auth requests to the backend server
export async function GET(request: NextRequest) {
  const url = new URL(request.url)
  const pathname = url.pathname.replace('/api/auth', '')
  const backendUrl = `http://localhost:8000/api/auth${pathname}${url.search}`
  
  console.log('Proxying GET to:', backendUrl)
  
  const response = await fetch(backendUrl, {
    headers: {
      ...Object.fromEntries(request.headers.entries()),
      host: 'localhost:8000',
    },
  })
  
  // If it's a redirect response, we need to handle it specially
  if (response.status >= 300 && response.status < 400) {
    const location = response.headers.get('location')
    if (location) {
      // Update redirect URLs to use port 3001 instead of 8000
      let newLocation = location.replace('http://localhost:8000', 'http://localhost:3001')
      
      // If this is a successful magic link verification, use the callbackURL from the query
      if (pathname.includes('/magic-link/verify') && response.status === 302) {
        const callbackUrl = url.searchParams.get('callbackURL')
        if (callbackUrl) {
          // Use the provided callback URL
          newLocation = callbackUrl
        } else if (location === 'http://localhost:3001' || location === 'http://localhost:3001/') {
          // Default to dashboard if no specific callback
          newLocation = 'http://localhost:3001/dashboard'
        }
      }
      
      // Create NextResponse for proper cookie handling
      const nextResponse = NextResponse.redirect(new URL(newLocation, request.url))
      
      // Pass along all set-cookie headers from the backend
      const setCookieHeaders = response.headers.getSetCookie()
      setCookieHeaders.forEach(cookie => {
        nextResponse.headers.append('set-cookie', cookie)
      })
      
      return nextResponse
    }
  }
  
  const data = await response.text()
  
  // Create NextResponse with proper cookie handling
  const nextResponse = new NextResponse(data, {
    status: response.status,
    headers: new Headers(),
  })
  
  // Copy headers, excluding problematic encoding headers
  response.headers.forEach((value, key) => {
    if (key.toLowerCase() !== 'content-encoding' && 
        key.toLowerCase() !== 'content-length' &&
        key.toLowerCase() !== 'transfer-encoding' &&
        key.toLowerCase() !== 'set-cookie') {
      nextResponse.headers.set(key, value)
    }
  })
  
  // Handle cookies properly
  const setCookieHeaders = response.headers.getSetCookie()
  setCookieHeaders.forEach(cookie => {
    nextResponse.headers.append('set-cookie', cookie)
  })
  
  return nextResponse
}

export async function POST(request: NextRequest) {
  const url = new URL(request.url)
  const pathname = url.pathname.replace('/api/auth', '')
  const backendUrl = `http://localhost:8000/api/auth${pathname}${url.search}`
  
  console.log('Proxying POST to:', backendUrl)
  
  const body = await request.text()
  
  const response = await fetch(backendUrl, {
    method: 'POST',
    headers: {
      ...Object.fromEntries(request.headers.entries()),
      host: 'localhost:8000',
    },
    body: body,
  })
  
  const data = await response.text()
  
  // Create NextResponse with proper cookie handling
  const nextResponse = new NextResponse(data, {
    status: response.status,
    headers: new Headers(),
  })
  
  // Copy headers, excluding problematic encoding headers
  response.headers.forEach((value, key) => {
    if (key.toLowerCase() !== 'content-encoding' && 
        key.toLowerCase() !== 'content-length' &&
        key.toLowerCase() !== 'transfer-encoding' &&
        key.toLowerCase() !== 'set-cookie') {
      nextResponse.headers.set(key, value)
    }
  })
  
  // Handle cookies properly
  const setCookieHeaders = response.headers.getSetCookie()
  setCookieHeaders.forEach(cookie => {
    nextResponse.headers.append('set-cookie', cookie)
  })
  
  return nextResponse
}

export async function PUT(request: NextRequest) {
  return POST(request)
}

export async function DELETE(request: NextRequest) {
  const url = new URL(request.url)
  const pathname = url.pathname.replace('/api/auth', '')
  const backendUrl = `http://localhost:8000/api/auth${pathname}${url.search}`
  
  const response = await fetch(backendUrl, {
    method: 'DELETE',
    headers: {
      ...Object.fromEntries(request.headers.entries()),
      host: 'localhost:8000',
    },
  })
  
  const data = await response.text()
  
  // Create NextResponse with proper cookie handling
  const nextResponse = new NextResponse(data, {
    status: response.status,
    headers: new Headers(),
  })
  
  // Copy headers, excluding problematic encoding headers
  response.headers.forEach((value, key) => {
    if (key.toLowerCase() !== 'content-encoding' && 
        key.toLowerCase() !== 'content-length' &&
        key.toLowerCase() !== 'transfer-encoding' &&
        key.toLowerCase() !== 'set-cookie') {
      nextResponse.headers.set(key, value)
    }
  })
  
  // Handle cookies properly
  const setCookieHeaders = response.headers.getSetCookie()
  setCookieHeaders.forEach(cookie => {
    nextResponse.headers.append('set-cookie', cookie)
  })
  
  return nextResponse
}