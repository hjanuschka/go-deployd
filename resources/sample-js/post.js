// Process data when creating new pets
// Available variables: this (data), me (user session), query, isRoot

// Auto-generate a unique identifier if not provided
if (!this.petId) {
    this.petId = 'PET-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
}

// Set default values
if (this.vaccinated === undefined) {
    this.vaccinated = false;
}

// Track creation metadata
this.createdBy = me ? me.id : 'anonymous';
this.status = 'active';

// Validate user permissions - only authenticated users or root can create
if (!me && !isRoot) {
    cancel('Authentication required to create pets');
}

// Log the creation
console.log('Creating new pet:', this.name, 'by user:', this.createdBy);

// Business logic - set adoption status
if (this.owner && this.owner.trim() !== '') {
    this.adoptionStatus = 'adopted';
    if (!this.adoptionDate) {
        this.adoptionDate = new Date().toISOString();
    }
} else {
    this.adoptionStatus = 'available';
}

// Protect sensitive fields from being exposed in API responses
protect('createdBy');