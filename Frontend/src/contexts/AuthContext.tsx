import React, { createContext, useContext, useReducer, useEffect, ReactNode } from 'react';
import { api, User, userAPI, tokenUtils } from '../services/api';
import websocketService from '../services/websocket';

interface AuthState {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

interface AuthContextType extends AuthState {
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => void;
}

type AuthAction =
  | { type: 'LOGIN_START' }
  | { type: 'LOGIN_SUCCESS'; payload: { user: User | null; token: string } }
  | { type: 'LOGIN_FAILURE' }
  | { type: 'LOGOUT' }
  | { type: 'REGISTER_START' }
  | { type: 'REGISTER_SUCCESS'; payload: { user: User; token: string } }
  | { type: 'REGISTER_FAILURE' };

const getInitialToken = (): string | null => {
  const token = localStorage.getItem('token');
  if (!token) {
    return null;
  }

  if (tokenUtils.isTokenExpired(token)) {
    tokenUtils.clearToken();
    return null;
  }

  api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
  return token;
};

const initialToken = getInitialToken();

const initialState: AuthState = {
  user: null,
  token: initialToken,
  isLoading: false,
  isAuthenticated: Boolean(initialToken),
};

const authReducer = (state: AuthState, action: AuthAction): AuthState => {
  switch (action.type) {
    case 'LOGIN_START':
    case 'REGISTER_START':
      return {
        ...state,
        isLoading: true,
      };
    case 'LOGIN_SUCCESS':
    case 'REGISTER_SUCCESS':
      return {
        ...state,
        user: action.payload.user,
        token: action.payload.token,
        isLoading: false,
        isAuthenticated: true,
      };
    case 'LOGIN_FAILURE':
    case 'REGISTER_FAILURE':
      return {
        ...state,
        isLoading: false,
      };
    case 'LOGOUT':
      return {
        ...state,
        user: null,
        token: null,
        isAuthenticated: false,
      };
    default:
      return state;
  }
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [state, dispatch] = useReducer(authReducer, initialState);

  useEffect(() => {
    const initializeAuth = async () => {
      const validToken = await tokenUtils.getValidToken();
      if (validToken) {
        try {
          // Fetch user data
          const user = await userAPI.getCurrentUser();
          dispatch({
            type: 'LOGIN_SUCCESS',
            payload: {
              user: user,
              token: validToken,
            },
          });
        } catch {
          // If fetching user fails, clear the invalid token
          tokenUtils.clearToken();
          dispatch({ type: 'LOGIN_FAILURE' });
        }
      } else {
        // No valid token, set loading to false
        dispatch({ type: 'LOGIN_FAILURE' });
      }
    };

    initializeAuth();
  }, []);

  const login = async (username: string, password: string) => {
    dispatch({ type: 'LOGIN_START' });
    try {
      const response = await api.post('/auth/login', { username, password });
      const { user, token } = response.data.data;
      
      tokenUtils.setToken(token);
      
      dispatch({
        type: 'LOGIN_SUCCESS',
        payload: { user, token },
      });
    } catch (error) {
      dispatch({ type: 'LOGIN_FAILURE' });
      throw error;
    }
  };

  const register = async (username: string, email: string, password: string) => {
    dispatch({ type: 'REGISTER_START' });
    try {
      const response = await api.post('/auth/register', { username, email, password });
      const { user, token } = response.data.data;
      
      tokenUtils.setToken(token);
      
      dispatch({
        type: 'REGISTER_SUCCESS',
        payload: { user, token },
      });
    } catch (error) {
      dispatch({ type: 'REGISTER_FAILURE' });
      throw error;
    }
  };

  const logout = () => {
    tokenUtils.clearToken();
    dispatch({ type: 'LOGOUT' });
  };

  useEffect(() => {
    if (state.token) {
      api.defaults.headers.common['Authorization'] = `Bearer ${state.token}`;
      // Connect WebSocket when user is authenticated
      if (state.user?.role === 'admin') {
        websocketService.connect();
      }
    } else {
      // Disconnect WebSocket when user is not authenticated
      websocketService.disconnect();
    }
  }, [state.token, state.user]);

  useEffect(() => {
    if (state.token) {
      api.defaults.headers.common['Authorization'] = `Bearer ${state.token}`;
    }
  }, [state.token]);

  return (
    <AuthContext.Provider
      value={{
        ...state,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
