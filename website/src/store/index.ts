import { combineReducers, configureStore } from '@reduxjs/toolkit'
import authSlice, { AuthState } from '../features/auth'
import storage from 'redux-persist/lib/storage'
import { setupListeners } from '@reduxjs/toolkit/query'

import {
  persistReducer,
  FLUSH,
  REHYDRATE,
  PAUSE,
  PERSIST,
  PURGE,
  REGISTER,
  PersistConfig,
} from 'redux-persist'

const persistConfig: PersistConfig<any> = {
  key: 'root',
  storage: storage,
  whitelist: ['auth'],
  blacklist: [],
}

export type RootState = {
  auth: AuthState
}

export const rootReducers = combineReducers<RootState>({
  auth: authSlice.reducer,
})

const persistedReducer = persistReducer<RootState>(persistConfig, rootReducers)

export const store = configureStore({
  reducer: persistedReducer,
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        ignoredActions: [FLUSH, REHYDRATE, PAUSE, PERSIST, PURGE, REGISTER],
      },
    }),
})

setupListeners(store.dispatch)

export type AppDispatch = typeof store.dispatch
