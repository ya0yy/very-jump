import React from 'react';

type ErrorBoundaryState = { hasError: boolean; error?: Error };

class ErrorBoundary extends React.Component<React.PropsWithChildren, ErrorBoundaryState> {
  constructor(props: React.PropsWithChildren) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // 简单上报到控制台，便于排查
    // eslint-disable-next-line no-console
    console.error('React ErrorBoundary捕获错误:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: 24 }}>
          <h2>页面渲染出错</h2>
          <p style={{ color: '#ff4d4f' }}>{this.state.error?.message}</p>
          <p>请刷新页面或重新登录。如果问题持续，请截图控制台错误信息反馈。</p>
        </div>
      );
    }
    return this.props.children;
  }
}

export default ErrorBoundary;










