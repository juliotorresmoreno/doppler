import React from 'react'
import { getGoogleUrl } from '../utils/getGoogleUrl'

const from = 'http://localhost:4080'

const GoogleLoginButton: React.FC = () => {
  return <a href={getGoogleUrl(from)}>Continue with Google</a>
}

export default GoogleLoginButton
