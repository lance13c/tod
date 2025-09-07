import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  webpack: (config, { isServer }) => {
    if (isServer) {
      // Externalize native modules and their optional dependencies
      config.externals = [...(config.externals || []), {
        'aws-sdk': 'commonjs aws-sdk',
        'mock-aws-s3': 'commonjs mock-aws-s3',
        'nock': 'commonjs nock',
        '@mapbox/node-pre-gyp': 'commonjs @mapbox/node-pre-gyp',
        'duckdb': 'commonjs duckdb',
        'duckdb-async': 'commonjs duckdb-async',
      }];
    }

    // Ignore non-JS files that webpack shouldn't process
    config.module = {
      ...config.module,
      rules: [
        ...(config.module?.rules || []),
        {
          test: /\.(html|cs)$/,
          use: 'ignore-loader',
        },
      ],
    };

    return config;
  },
  // Disable output file tracing for native modules
  outputFileTracingExcludes: {
    '*': [
      'node_modules/duckdb/**/*',
      'node_modules/@mapbox/**/*',
    ],
  },
  // External packages that should not be bundled
  serverExternalPackages: [
    'duckdb',
    'duckdb-async',
    '@mapbox/node-pre-gyp',
  ],
};

export default nextConfig;
