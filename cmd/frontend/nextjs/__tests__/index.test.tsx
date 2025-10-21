import { render, screen } from '@testing-library/react';
import Home from '../pages/index';

// Mock Next.js router
jest.mock('next/router', () => ({
  useRouter() {
    return {
      route: '/',
      pathname: '/',
      query: {},
      asPath: '/',
    };
  },
}));

describe('Home Page', () => {
  it('renders the main heading', () => {
    render(<Home />);
    const heading = screen.getByText('ExplainIQ');
    expect(heading).toBeInTheDocument();
  });

  it('renders the topic input form', () => {
    render(<Home />);
    const input = screen.getByPlaceholderText(/what would you like to learn about/i);
    expect(input).toBeInTheDocument();
  });

  it('renders the start learning button', () => {
    render(<Home />);
    const button = screen.getByText('Start Learning');
    expect(button).toBeInTheDocument();
  });
});



