// ============================================================================
// DARK MODE TOGGLE
// ============================================================================

function toggleDarkMode() {
  const isDark = localStorage.getItem('darkMode') === 'true';
  localStorage.setItem('darkMode', !isDark);
  applyDarkMode(!isDark);
}

function applyDarkMode(isDark) {
  if (isDark) {
    document.documentElement.style.colorScheme = 'dark';
  } else {
    document.documentElement.style.colorScheme = 'light';
  }
}

// ============================================================================
// COMMENT FORM HANDLER
// ============================================================================

document.addEventListener('DOMContentLoaded', function() {
  const commentForm = document.getElementById('commentForm');
  if (commentForm) {
    commentForm.addEventListener('submit', async function(e) {
      e.preventDefault();

      const formData = {
        author_name: document.getElementById('name').value,
        author_email: document.getElementById('email').value,
        author_url: document.getElementById('url').value,
        content: document.getElementById('content').value,
      };

      const postID = new URLSearchParams(window.location.search).get('post_id');

      try {
        const response = await fetch(`/api/comments?post_id=${postID}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(formData),
        });

        if (response.ok) {
          alert('Comment submitted! It will appear after approval.');
          commentForm.reset();
        } else {
          alert('Failed to submit comment. Please try again.');
        }
      } catch (error) {
        console.error('Error:', error);
        alert('An error occurred. Please try again.');
      }
    });
  }
});

// ============================================================================
// SEARCH FUNCTIONALITY
// ============================================================================

function performSearch(query) {
  if (query.length < 2) {
    alert('Search query must be at least 2 characters long');
    return;
  }
  window.location.href = `/search?q=${encodeURIComponent(query)}`;
}

// ============================================================================
// ADMIN UTILITIES
// ============================================================================

async function deleteItem(type, id, confirmMessage = 'Are you sure?') {
  if (!confirm(confirmMessage)) return;

  try {
    const response = await fetch(`/admin/${type}/${id}/delete`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (response.ok) {
      location.reload();
    } else {
      alert('Failed to delete item');
    }
  } catch (error) {
    console.error('Error:', error);
    alert('An error occurred');
  }
}

async function approveComment(id) {
  try {
    const response = await fetch(`/admin/comments/${id}/approve`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (response.ok) {
      location.reload();
    } else {
      alert('Failed to approve comment');
    }
  } catch (error) {
    console.error('Error:', error);
    alert('An error occurred');
  }
}

// ============================================================================
// FORM HELPERS
// ============================================================================

function initMDEditor(elementId) {
  const element = document.getElementById(elementId);
  if (element) {
    // Simple markdown preview enhancement
    element.addEventListener('input', function() {
      // Could integrate a markdown preview panel here
    });
  }
}

async function uploadFile(file, onSuccess) {
  const formData = new FormData();
  formData.append('file', file);

  try {
    const response = await fetch('/admin/upload', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: formData,
    });

    const data = await response.json();
    if (response.ok) {
      onSuccess(data.url);
    } else {
      alert('Upload failed: ' + data.error);
    }
  } catch (error) {
    console.error('Error:', error);
    alert('An error occurred during upload');
  }
}

// ============================================================================
// AUTHENTICATION HELPERS
// ============================================================================

function login(email, password) {
  const loginData = { email, password };

  return fetch('/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(loginData),
  })
  .then(response => response.json())
  .then(data => {
    if (data.token) {
      localStorage.setItem('token', data.token);
      localStorage.setItem('user', JSON.stringify(data.user));
      return data;
    } else {
      throw new Error(data.error || 'Login failed');
    }
  });
}

function logout() {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
  window.location.href = '/';
}

function getAuthToken() {
  return localStorage.getItem('token');
}

function getCurrentUser() {
  const user = localStorage.getItem('user');
  return user ? JSON.parse(user) : null;
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

function formatDate(dateString) {
  const options = { year: 'numeric', month: 'long', day: 'numeric' };
  return new Date(dateString).toLocaleDateString(undefined, options);
}

function truncate(text, length) {
  if (text.length <= length) return text;
  return text.substring(0, length) + '...';
}

// ============================================================================
// INITIALIZE ON PAGE LOAD
// ============================================================================

document.addEventListener('DOMContentLoaded', function() {
  // Apply saved dark mode preference
  const isDark = localStorage.getItem('darkMode') === 'true';
  if (isDark) {
    applyDarkMode(true);
  }

  // Initialize markdown editor if present
  const mdEditor = document.getElementById('markdown-editor');
  if (mdEditor) {
    initMDEditor('markdown-editor');
  }
});
