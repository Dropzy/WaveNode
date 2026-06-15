# Music Player Frontend

A Spotify-like music player web application built with React, TypeScript, and Vite. This frontend connects to the Go backend API to provide a complete music streaming experience.

## Features

- **Spotify-like UI**: Dark theme with green accents, matching Spotify's design language
- **Music Library**: Browse and search through your music collection
- **Playlist Management**: Create, view, and manage playlists
- **Search Functionality**: Search for tracks by title, artist, album, or genre
- **Responsive Design**: Works on desktop and mobile devices
- **Modern Stack**: Built with React 18, TypeScript, and Styled Components

## Tech Stack

- **React 18** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **React Router** - Client-side routing
- **Styled Components** - CSS-in-JS styling
- **Axios** - HTTP client for API calls
- **Lucide React** - Icon library

## Prerequisites

- Node.js 16+ and npm
- The Go backend server running on `http://localhost:8080`

## Installation

1. Navigate to the Frontend directory:
```bash
cd Frontend
```

2. Install dependencies:
```bash
npm install
```

## Development

Start the development server:
```bash
npm run dev
```

The application will be available at `http://localhost:5173` (or the next available port).

## Building for Production

Create a production build:
```bash
npm run build
```

Preview the production build:
```bash
npm run preview
```

## Project Structure

```
Frontend/
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── Layout.tsx      # Main layout wrapper
│   │   ├── Sidebar.tsx     # Navigation sidebar
│   │   └── Player.tsx      # Music player controls
│   ├── pages/              # Page components
│   │   ├── Home.tsx        # Home page with featured content
│   │   ├── Library.tsx     # Music library view
│   │   ├── Search.tsx      # Search results page
│   │   └── Playlist.tsx    # Individual playlist view
│   ├── services/           # API services
│   │   └── api.ts          # API client and types
│   ├── styles/             # Global styles
│   │   └── GlobalStyle.ts  # Global styled-components
│   ├── App.tsx             # Main app component with routing
│   └── main.tsx            # App entry point
├── public/                 # Static assets
├── index.html              # HTML template
├── package.json            # Dependencies and scripts
└── vite.config.ts          # Vite configuration
```

## API Integration

The frontend connects to the Go backend API at `http://localhost:8080`. The API endpoints include:

- `GET /api/music` - Get all music tracks
- `GET /api/music/search?q={query}` - Search music
- `GET /api/playlists` - Get all playlists
- `GET /api/playlists/{id}` - Get specific playlist
- `POST /api/playlists` - Create new playlist
- And more...

## Usage

1. **Home Page**: View featured playlists and recently played tracks
2. **Library**: Browse your complete music collection organized by albums and artists
3. **Search**: Find specific tracks, albums, or artists
4. **Playlists**: View and manage your playlists
5. **Player**: Control music playback with the bottom player bar

## Styling

The application uses Styled Components for styling with a Spotify-inspired design:

- **Dark Theme**: Black background (#000) with gray accents
- **Primary Color**: Spotify green (#1db954)
- **Typography**: System fonts for optimal performance
- **Responsive**: Adapts to different screen sizes

## Development Notes

- The app uses React 18 with concurrent features
- TypeScript provides type safety throughout the application
- Styled Components allow for dynamic, component-based styling
- The API service layer handles all backend communication
- React Router manages client-side navigation

## Troubleshooting

1. **API Connection Issues**: Ensure the backend server is running on port 8080
2. **Build Errors**: Check that all dependencies are installed with `npm install`
3. **Port Conflicts**: Vite will automatically use the next available port if 5173 is in use

## Future Enhancements

- [ ] Audio playback functionality
- [ ] User authentication
- [ ] Real-time updates
- [ ] Offline support
- [ ] Advanced audio controls (equalizer, crossfade)
- [ ] Social features (sharing, following)
- [ ] Podcast support
