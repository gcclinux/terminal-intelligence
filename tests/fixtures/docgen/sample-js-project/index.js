/**
 * Sample JavaScript module for testing documentation generation
 */

/**
 * Print a greeting message
 */
function helloWorld() {
    console.log("Hello, World!");
}

/**
 * A class for greeting people
 */
class Greeter {
    /**
     * Create a greeter
     * @param {string} name - The name to greet
     */
    constructor(name) {
        this.name = name;
    }
    
    /**
     * Print a personalized greeting
     */
    greet() {
        console.log(`Hello, ${this.name}!`);
    }
}

module.exports = { helloWorld, Greeter };
