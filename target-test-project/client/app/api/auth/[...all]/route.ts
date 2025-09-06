import { NextRequest } from 'next/server'

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
      
      // If this is a successful magic link verification, redirect to dashboard
      if (pathname.includes('/magic-link/verify') && response.status === 302) {
        // Check if the redirect is going to the callback URL
        const locationUrl = new URL(location, 'http://localhost:3001')
        if (locationUrl.pathname === '/' || locationUrl.pathname === '') {
          newLocation = 'http://localhost:3001/dashboard'
        }
      }
      
      // Pass along any set-cookie headers from the backend
      const headers = new Headers()
      response.headers.forEach((value, key) => {
        // Only pass through set-cookie headers for redirects
        if (key.toLowerCase() === 'set-cookie') {
          headers.append(key, value)
        }
      })
      headers.set('location', newLocation)
      
      return new Response(null, {
        status: response.status,
        headers,
      })
    }
  }
  
  const data = await response.text()
  
  // Create response with proper headers including cookies
  const responseHeaders = new Headers()
  response.headers.forEach((value, key) => {
    // Skip content-encoding and content-length headers as we're returning decoded text
    if (key.toLowerCase() !== 'content-encoding' && 
        key.toLowerCase() !== 'content-length' &&
        key.toLowerCase() !== 'transfer-encoding') {
      responseHeaders.append(key, value)
    }
  })
  
  return new Response(data, {
    status: response.status,
    headers: responseHeaders,
  })
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
  
  // Create response with proper headers including cookies
  const responseHeaders = new Headers()
  response.headers.forEach((value, key) => {
    // Skip content-encoding and content-length headers as we're returning decoded text
    if (key.toLowerCase() !== 'content-encoding' && 
        key.toLowerCase() !== 'content-length' &&
        key.toLowerCase() !== 'transfer-encoding') {
      responseHeaders.append(key, value)
    }
  })
  
  return new Response(data, {
    status: response.status,
    headers: responseHeaders,
  })
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
  
  // Create response with proper headers including cookies
  const responseHeaders = new Headers()
  response.headers.forEach((value, key) => {
    // Skip content-encoding and content-length headers as we're returning decoded text
    if (key.toLowerCase() !== 'content-encoding' && 
        key.toLowerCase() !== 'content-length' &&
        key.toLowerCase() !== 'transfer-encoding') {
      responseHeaders.append(key, value)
    }
  })
  
  return new Response(data, {
    status: response.status,
    headers: responseHeaders,
  })
}