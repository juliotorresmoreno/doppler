import { createAsyncThunk, createSlice } from '@reduxjs/toolkit'
import { ISession } from '../models/session'
import config from '../config'

export type AuthState = {
  session: ISession | null
  isLoading: boolean
  error: string | null
}

const initialState: AuthState = {
  session: null,
  isLoading: false,
  error: null,
}

export const signIn = createAsyncThunk('auth/sign-in', async (payload: any) => {
  const res = await fetch(config.baseUrl + '/auth/sign-in', {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
    },
    credentials: 'same-origin',
    mode: 'cors',
    body: JSON.stringify(payload),
  })
  const data = await res.json()
  if (!res.ok) {
    throw new Error(data.message)
  }
  return data
})

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    logout(state) {
      state.session = null
    },
  },
  extraReducers: (builder) => {
    builder.addCase(signIn.pending, (state) => {
      state.error = null
      state.isLoading = true
    })
    builder.addCase(signIn.fulfilled, (state, action) => {
      state.isLoading = false
      state.session = action.payload
    })
    builder.addCase(signIn.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message ?? null
    })
  },
})

export default authSlice
