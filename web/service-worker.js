const CACHE_NAME = 'giggum-pwa-v2';
const API_CACHE_NAME = 'giggum-api-v1';

// Core app resources to cache
const urlsToCache = [
  '/',
  '/static/styles.css',
  '/static/app.js',
  '/manifest.json'
];

// API endpoints to cache for offline access
const apiUrlsToCache = [
  '/api/agents',
  '/api/projects',
  '/api/sessions',
  '/api/notifications'
];

// Install event - cache resources
self.addEventListener('install', event => {
  event.waitUntil(
    Promise.all([
      // Cache core app resources
      caches.open(CACHE_NAME)
        .then(cache => {
          console.log('Opened core cache');
          return cache.addAll(urlsToCache);
        }),
      // Cache API endpoints
      caches.open(API_CACHE_NAME)
        .then(cache => {
          console.log('Opened API cache');
          return cache.addAll(apiUrlsToCache.map(url => new Request(url, { method: 'GET' })));
        })
    ])
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.map(cacheName => {
          if (cacheName !== CACHE_NAME && cacheName !== API_CACHE_NAME) {
            console.log('Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
});

// Fetch event - serve from cache when offline
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);
  
  // Handle API requests
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(handleApiRequest(event.request));
    return;
  }
  
  // Handle other requests (static files, etc.)
  event.respondWith(handleCoreRequest(event.request));
});

// Handle API requests with cache-first strategy
async function handleApiRequest(request) {
  const url = new URL(request.url);
  
  try {
    // Try cache first
    const cachedResponse = await caches.match(request, { cacheName: API_CACHE_NAME });
    if (cachedResponse) {
      // Update cache in background
      updateApiCache(request);
      return cachedResponse;
    }
    
    // Fetch from network
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(API_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
    
  } catch (error) {
    console.log('API request failed, trying cache:', request.url);
    
    // Fallback to cache
    const cachedResponse = await caches.match(request, { cacheName: API_CACHE_NAME });
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // If still no cache, try to return cached index for offline fallback
    const indexResponse = await caches.match('/', { cacheName: CACHE_NAME });
    if (indexResponse && request.method === 'GET') {
      return new Response(
        JSON.stringify({ 
          error: 'Offline - no cached data available',
          offline: true,
          endpoint: url.pathname
        }),
        {
          status: 503,
          statusText: 'Service Unavailable',
          headers: { 'Content-Type': 'application/json' }
        }
      );
    }
    
    throw error;
  }
}

// Handle core requests with cache-first strategy
async function handleCoreRequest(request) {
  try {
    // Try cache first
    const cachedResponse = await caches.match(request, { cacheName: CACHE_NAME });
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // Fetch from network
    const networkResponse = await fetch(request);
    
    if (networkResponse.ok) {
      const cache = await caches.open(CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
    
    return networkResponse;
    
  } catch (error) {
    console.log('Core request failed, trying cache:', request.url);
    
    // Fallback to cache
    const cachedResponse = await caches.match(request, { cacheName: CACHE_NAME });
    if (cachedResponse) {
      return cachedResponse;
    }
    
    // For navigation requests, fallback to index page
    if (request.mode === 'navigate') {
      const indexResponse = await caches.match('/', { cacheName: CACHE_NAME });
      if (indexResponse) {
        return indexResponse;
      }
    }
    
    throw error;
  }
}

// Update API cache in background
async function updateApiCache(request) {
  try {
    const networkResponse = await fetch(request);
    if (networkResponse.ok) {
      const cache = await caches.open(API_CACHE_NAME);
      cache.put(request, networkResponse.clone());
    }
  } catch (error) {
    console.log('Failed to update API cache:', error);
  }
}

// Activate event - clean up old caches
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return Promise.all(
        cacheNames.map(cacheName => {
          if (cacheName !== CACHE_NAME) {
            console.log('Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
});