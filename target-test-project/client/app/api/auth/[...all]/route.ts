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
      const newLocation = location.replace('http://localhost:8000', 'http://localhost:3001')
      return new Response(null, {
        status: response.status,
        headers: {
          ...Object.fromEntries(response.headers.entries()),
          location: newLocation,
        },
      })
    }
  }
  
  const data = await response.text()
  
  return new Response(data, {
    status: response.status,
    headers: response.headers,
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
  
  return new Response(data, {
    status: response.status,
    headers: response.headers,
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
  
  return new Response(data, {
    status: response.status,
    headers: response.headers,
  })
}