// Validate pet data before saving
// Available variables: this (data), me (user session), query, isRoot

// Validate required fields
if (!this.name || this.name.trim() === '') {
    error('name', 'Pet name is required');
}

if (!this.breed || this.breed.trim() === '') {
    error('breed', 'Pet breed is required');
}

if (!this.age || this.age < 0) {
    error('age', 'Pet age must be a positive number');
}

// Validate optional fields
if (this.weight && this.weight <= 0) {
    error('weight', 'Weight must be greater than 0');
}

if (this.age && this.age > 30) {
    error('age', 'Pet age seems unrealistic (max 30 years)');
}

// Normalize data
if (this.name) {
    this.name = this.name.trim();
}

if (this.breed) {
    this.breed = this.breed.trim();
}

if (this.color) {
    this.color = this.color.trim().toLowerCase();
}

if (this.owner) {
    this.owner = this.owner.trim();
}

// Access control - only authenticated users can create pets
if (!me) {
    cancel('Authentication required to manage pets');
}

console.log('Validating pet:', this.name, 'breed:', this.breed);