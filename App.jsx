import React from 'react';

function App() {
  return (
    <div style={styles.page}>
      <div style={styles.card}>
        <p style={styles.eyebrow}>Live Polling Tool</p>
        <h1 style={styles.heading}>Create and manage live polls in seconds</h1>
        <p style={styles.description}>
          Gather instant feedback from your audience with real-time voting, simple
          setup, and powerful insights for any event or discussion.
        </p>

        <div style={styles.actions}>
          <button type="button" style={{ ...styles.button, ...styles.primaryButton }}>
            Create Poll
          </button>
          <button type="button" style={{ ...styles.button, ...styles.secondaryButton }}>
            Log In
          </button>
        </div>
      </div>
    </div>
  );
}

const styles = {
  page: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'linear-gradient(135deg, #f5f7ff 0%, #eef4ff 100%)',
    padding: '24px',
    fontFamily: 'Arial, sans-serif',
  },
  card: {
    maxWidth: '640px',
    width: '100%',
    background: '#ffffff',
    borderRadius: '20px',
    boxShadow: '0 18px 45px rgba(15, 23, 42, 0.08)',
    padding: '48px 40px',
    textAlign: 'center',
  },
  eyebrow: {
    margin: '0 0 16px',
    color: '#4f46e5',
    fontSize: '14px',
    fontWeight: 700,
    letterSpacing: '0.12em',
    textTransform: 'uppercase',
  },
  heading: {
    margin: '0 0 18px',
    fontSize: 'clamp(2rem, 5vw, 3.5rem)',
    lineHeight: 1.1,
    color: '#111827',
  },
  description: {
    margin: '0 auto 32px',
    maxWidth: '540px',
    fontSize: '1.05rem',
    lineHeight: 1.6,
    color: '#4b5563',
  },
  actions: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '16px',
    flexWrap: 'wrap',
  },
  button: {
    border: 'none',
    borderRadius: '12px',
    padding: '14px 24px',
    fontSize: '1rem',
    fontWeight: 700,
    cursor: 'pointer',
    transition: 'transform 0.2s ease, box-shadow 0.2s ease',
  },
  primaryButton: {
    background: '#4f46e5',
    color: '#ffffff',
    boxShadow: '0 10px 20px rgba(79, 70, 229, 0.25)',
  },
  secondaryButton: {
    background: '#eef2ff',
    color: '#1f2937',
  },
};

export default App;