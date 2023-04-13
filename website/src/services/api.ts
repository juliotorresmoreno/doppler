import React, { useState } from 'react'
import { useAppSelector } from '../store/hooks'
import { HTTPError } from '../types/http'

type ApiOpts = {}

export function useGetData<Response = any>(url: string, opts?: ApiOpts) {
  const { error, isLoading, apply } = useApi<null, Response>('GET', url, opts)

  return { isLoading, error, get: () => apply(null) }
}

function applyOptsToRequest<T = any>(url: string, args: RequestInit, payload: T) {
  if (!payload) return url

  if (payload instanceof FormData) {
    const id: string = payload.get('id')?.toString() ?? ''
    args.body = payload
    if (id) {
      payload.delete('id')
      return [url, id].join('/')
    }
    return url
  }

  const id: string = (payload as any).id ?? ''
  args.body = JSON.stringify(payload)
  args.headers = {
    ...args.headers,
    'Content-Type': 'application/json'
  }
  if (id) {
    delete (payload as any).id
    return [url, id].join('/')
  }
  return url
}

export function useApi<Payload = any, Response = any>(
  method: string,
  url: string,
  opts: ApiOpts = {}
) {
  const session = useAppSelector((state: any) => state.auth.session)
  const [isLoading, setIsLoading] = useState<boolean>(false)
  const [error, setError] = useState<HTTPError | null>(null)

  const apply = async (payload: Payload) => {
    setIsLoading(true)
    setError(null)

    try {
      const args: RequestInit = {
        method: method,
        headers: {
          'X-API-Key': session?.token ?? '',
        },
      }
      const _url = applyOptsToRequest(url, args, payload)

      const response = await fetch(_url, args)
      const content = await response.json()
      if (!response.ok) {
        throw new Error(content.message)
      }
      setIsLoading(false)
      return content as Response
    } catch (err) {
      setIsLoading(false)
      setError({
        message: (err as Error).message,
      })
      throw err
    }
  }

  return { isLoading, error, apply }
}

export function useAdd<Payload = any, Response = any>(
  url: string,
  opts?: ApiOpts
) {
  const { error, isLoading, apply } = useApi<Payload, Response>(
    'POST',
    url,
    opts
  )

  return { isLoading, error, add: apply }
}

type WithId = { id: string | number }

export function useUpdate<Payload = any, Response = any>(
  url: string,
  opts?: ApiOpts
) {
  type PayloadWithId = Payload & WithId
  const { error, isLoading, apply } = useApi<PayloadWithId, Response>(
    'PATCH',
    url,
    opts
  )

  return { isLoading, error, update: apply }
}

export function useRemove(url: string, opts?: ApiOpts) {
  const _url = url
  const { error, isLoading, apply } = useApi<WithId | null, void>(
    'DELETE',
    _url,
    opts
  )

  return {
    isLoading,
    error,
    remove: async (id: string | number | null) => {
      return apply(id !== null ? { id } : null)
    },
  }
}
