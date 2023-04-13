const path = require('path')
const HTMLWebpackPlugin = require('html-webpack-plugin')
const Dotenv = require('dotenv-webpack')
const isDev = process.env.NODE_ENV !== 'production'

/**
 * @typedef {import('webpack').WebpackOptionsNormalized} WebpackOptionsNormalized
 */

/**
 * @type {WebpackOptionsNormalized}
 */
const configuration = {
  entry: path.resolve(__dirname, './src/index.ts'),
  output: {
    filename: 'app.bundle.js',
    publicPath: '/',
  },
  plugins: [
    new HTMLWebpackPlugin({
      template: 'src/index.html',
    }),
    new Dotenv(),
  ],
  devtool: isDev ? 'source-map' : false,
  mode: isDev ? 'development' : 'production',
  cache: false,
  devServer: {
    compress: true,
    port: 1250,
    host: '127.0.0.1',
    static: path.resolve(__dirname, './public'),
  },
  module: {
    rules: [
      {
        test: /\.(js|jsx|ts|tsx)$/,
        exclude: /node_modules/,
        use: {
          loader: 'ts-loader',
        },
      },
      {
        test: /\.(css|scss)$/i,
        use: [
          // Creates `style` nodes from JS strings
          'style-loader',
          // Translates CSS into CommonJS
          'css-loader',
          // Compiles Sass to CSS
          'sass-loader',
        ],
      },
    ],
  },
  resolve: {
    extensions: ['.js', '.jsx', '.ts', '.tsx', '.css', 'scss'],
  },
}

module.exports = configuration
