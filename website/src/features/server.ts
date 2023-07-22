import { createAsyncThunk, createSlice } from '@reduxjs/toolkit'
import config from '../config'
import { store } from '../store'

export type AuthState = {
  cache: { [x: number]: any }
  isLoading: boolean
  error: string | null
}

const initialState: AuthState = {
  cache: {},
  isLoading: false,
  error: null,
}

export const get = createAsyncThunk('auth/get', async (id: number) => {
  const token = store.getState().auth.session?.token
  const res = await fetch(config.baseUrl + '/server/' + id, {
    method: 'GET',
    headers: {
      authorization: 'Bearer ' + token,
    },
    credentials: 'same-origin',
    mode: 'cors',
  })
  const data = await res.json()
  if (!res.ok) {
    throw new Error(data.message)
  }
  return data
})

const serverSlice = createSlice({
  name: 'server',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder.addCase(get.pending, (state) => {
      state.error = null
      state.isLoading = true
    })
    builder.addCase(get.fulfilled, (state, action) => {
      state.isLoading = false
      //state.cache = action.payload
      console.log(action)
    })
    builder.addCase(get.rejected, (state, action) => {
      state.isLoading = false
      state.error = action.error.message ?? null
    })
  },
})

export default serverSlice
