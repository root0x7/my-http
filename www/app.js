console.log('SimpleHTTP Server is running!');

// Simple JavaScript to show interactivity
document.addEventListener('DOMContentLoaded', function() {
    const title = document.querySelector('h1');
    if (title) {
        title.addEventListener('click', function() {
            title.style.color = title.style.color === 'red' ? '#2c3e50' : 'red';
        });
    }
});