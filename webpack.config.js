const path = require('path');
const TerserPlugin = require('terser-webpack-plugin');

module.exports = {
  module: {
    rules: [
      {
        test: /\.(sa|sc|c)ss$/, // 対象となるファイルの拡張子
        use: [
          "style-loader", // linkタグに出力する機能
          { // CSSをハンドルするための機能
            loader: "css-loader",
            options: {
              url: false // optionでcss内のurl()メソッドの取り込みを禁止する
            }
          },
          "sass-loader",
        ]
      }
    ]
  },
  entry: './src/app.js',
  output: {
    // 出力するファイル名
    filename: 'main.js',
    // 出力先のpathは絶対pathを指定する
    path: path.resolve(__dirname, 'static'),
  },
  devtool: 'inline-source-map',
  devServer: {
    host: 'localhost',
    contentBase: path.resolve(__dirname, 'static'),
    disableHostCheck: true,
    watchContentBase: true,
    port: 8080,
    open: false, // ブラウザを自動で立ち上げるかどうかを設定する
    proxy: {
      '/login': {
        target: 'http://localhost:3000'
      }
    }
  },
  // productionモードで有効になるoptimization.minmizerを上書きする
  optimization: {
    minimizer: [
      new TerserPlugin({
        terserOptions: {
          compress: { drop_console: true }
        }
      })
    ]
  }
};
