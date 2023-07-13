export default {
  title: process.env.APP_NAME,
  baseUrl: process.env.BASE_URL ?? document.location.href + 'api/v1',
  wsUrl: process.env.WS_URL ?? document.location.href + 'realtime',

  google_redirect_uri: process.env.GOOGLE_REDIRECT_URI ?? '',
  google_client_id: process.env.GOOGLE_CLIENT_ID ?? '',
}
