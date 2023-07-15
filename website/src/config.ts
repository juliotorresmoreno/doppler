const env = process.env
const ENV = env.NODE_ENV ?? 'development'

export default {
  env: ENV,

  title: env.APP_NAME ?? '',
  baseUrl: env.BASE_URL ?? document.location.href + 'api',
  wsUrl: env.WS_URL ?? document.location.href + 'realtime',

  testUserName: ENV === 'development' ? env.TEST_USER_NAME ?? '' : '',
  testUserLastname: ENV === 'development' ? env.TEST_USER_LASTNAME ?? '' : '',
  testUserEmail: ENV === 'development' ? env.TEST_USER_EMAIL ?? '' : '',
  testUserPassword: ENV === 'development' ? env.TEST_USER_PASSWORD ?? '' : '',

  google_redirect_uri: env.GOOGLE_REDIRECT_URI ?? '',
  google_client_id: env.GOOGLE_CLIENT_ID ?? '',
}
